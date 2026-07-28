// Commande server expose les surfaces HTTP du socle.
//
// # C'est le composition root : le SEUL code autorisé à tout connaître
//
// Il connaît la configuration, les pilotes, les adaptateurs et les modules. Tout
// le reste du dépôt ne connaît que des types fonction. C'est ce déséquilibre
// assumé qui garde le cœur pur : il faut bien qu'un endroit branche les fils, et
// cet endroit est ici, visible, plutôt que dispersé dans un conteneur
// d'injection (ADR 004).
//
// # Zéro prérequis d'infrastructure
//
// Avec la configuration livrée, ce binaire démarre sans base, sans Redis, sans
// Docker : tous les pilotes par défaut sont en mémoire. C'est la promesse de
// l'ADR 012, et ce fichier est l'endroit où elle se vérifie — `go run
// ./cmd/server` doit fonctionner sur une machine vierge.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/outboxpub"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"

	authhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/adapters/primary/http"
	userhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/http"
)

// Injectés à la compilation par la CI (voir Dockerfile).
//
// Globales assumées : `-ldflags -X` ne sait écrire que dans une variable de
// paquet. Il n'existe aucune autre façon de graver la version dans le binaire.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		// Écrit sur stderr et non par le logger : l'échec peut précéder la
		// construction du logger lui-même.
		fmt.Fprintf(os.Stderr, "démarrage impossible: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// NotifyContext avant tout le reste : un Ctrl+C pendant l'initialisation doit
	// interrompre, pas être avalé.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Le catalogue AVANT la configuration : c'est lui qui dit ce que la
	// configuration a le droit de nommer (ADR 014). Aucune table de modules ne
	// vit dans `internal/config` — elle y nommerait des modules, ce que la
	// règle 7 d'`arch-go` lui interdit.
	catalog, err := moduleCatalog()
	if err != nil {
		return fmt.Errorf("catalogue des modules: %w", err)
	}

	cfg, err := config.Load(catalog)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	logger := telemetry.NewLogger(cfg)

	// L'observabilité AVANT tout le reste : un défaut d'initialisation qui
	// surviendrait après le montage des modules ne serait tracé nulle part.
	arret, err := demarrerObservabilite(ctx, cfg)
	if err != nil {
		return err
	}
	defer arreterObservabilite(ctx, arret, logger)

	// Un contexte dérivé, pour que le serveur de métriques puisse couper le
	// service : sans lui, son échec resterait un avertissement dans un journal.
	ctx, couper := context.WithCancel(ctx)
	defer couper()
	servirMetriques(ctx, cfg, logger, couper)

	logger.InfoContext(ctx, "démarrage",
		slog.String("service", cfg.App.Name),
		slog.String("env", string(cfg.App.Env)),
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", buildDate),
	)

	return servir(ctx, cfg, logger)
}

// servir monte les surfaces et tient le serveur jusqu'à l'annulation.
//
// Extraite de `run` parce que celle-ci dépassait le seuil de lignes d'`arch-go`
// en y branchant l'observabilité. Le garde a attrapé la régression avant la
// relecture — c'est exactement ce qu'on lui demande.
//
// La coupure n'est pas arbitraire : `run` porte désormais l'AMORÇAGE — signaux,
// catalogue, configuration, journal, observabilité — et `servir` porte les
// SURFACES. Deux responsabilités, deux fonctions.
func servir(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	mounted, err := compose(cfg, logger)
	if err != nil {
		return err
	}

	router := httpserver.NewRouter(cfg, logger, probes(mounted))
	authhttp.Mount(router.API, mounted.auth)
	userhttp.Mount(router.API, mounted.users)
	userhttp.MountAvailability(router.API, mounted.users)

	// Les chemins annoncés sont ceux qui RÉPONDENT : huma sert le contrat sous
	// `/openapi.json` et `/openapi.yaml`, jamais sous `/openapi` nu — qui rend 404.
	// Annoncer un chemin qui ne répond pas envoie chercher une panne inexistante.
	logger.InfoContext(ctx, "surfaces montées",
		slog.String("docs", "/docs"),
		slog.String("openapi", "/openapi.json · /openapi.yaml"),
	)

	if err := httpserver.New(cfg, router.Mux, logger).Run(ctx); err != nil {
		return fmt.Errorf("serveur HTTP: %w", err)
	}
	return nil
}

// assembled rassemble les modules montés.
//
// Structure nommée plutôt que retours multiples : au troisième module, une
// fonction rendrait quatre valeurs, et « plus de trois retours = un type
// manquant » est une leçon déjà payée trois fois dans ce dépôt.
type assembled struct {
	outbox outbox.Module
	auth   auth.Module
	users  userregistration.Module
}

// compose branche les modules noyau puis les modules métier.
//
// L'ordre n'est pas arbitraire : un module métier consomme les ports du noyau,
// jamais l'inverse (ADR 012).
func compose(cfg config.Config, logger *slog.Logger) (assembled, error) {
	// Le pool est nil : aucun pilote par défaut n'en a besoin. Un pilote
	// `postgres` activé sans base REFUSE le démarrage plutôt que de tomber en
	// panne à la première requête.
	outboxMod, err := outbox.New(cfg.Modules[outbox.Name], outbox.Deps{Now: time.Now})
	if err != nil {
		return assembled{}, fmt.Errorf("module outbox: %w", err)
	}

	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  cfg.Security.Argon2.MemoryKiB,
		Iterations: cfg.Security.Argon2.Iterations,
		Threads:    cfg.Security.Argon2.Threads,
	})

	// Le hachage vient d'ICI et non du module : il est coûteux, paramétré, et son
	// réglage appartient à la configuration de sécurité de l'application. Un
	// module qui choisirait ses propres paramètres Argon2 les figerait pour tout
	// le monde — et le socle a précisément un endroit où ce réglage se décide.
	authMod, err := auth.New(cfg.Modules[auth.Name], auth.Deps{
		HashSecret:   hasher.Hash,
		VerifySecret: hasher.Verify,
		Now:          time.Now,
	})
	if err != nil {
		return assembled{}, fmt.Errorf("module auth: %w", err)
	}

	// Le pilote vient de la configuration, qui le porte enfin.
	//
	// Ce champ restait TOUJOURS vide avant l'ADR 014 : `config/modules.yaml` ne
	// validait que les modules noyau, donc y déclarer `user_registration` faisait
	// refuser le démarrage. Le module se rabattait silencieusement sur son
	// défaut, et la tranche de référence contournait le mécanisme qu'elle est
	// censée démontrer.
	driver := cfg.Modules.DriverOf(userregistration.Name)

	users, err := userregistration.New(driver, userregistration.Deps{
		HashPassword: hashing.New(hasher),
		PublishEvent: outboxpub.New(outboxMod.Enqueue),
		GenerateID:   generateUserID,
		Now:          userregistration.SystemClock(),
	})
	if err != nil {
		return assembled{}, fmt.Errorf("module user_registration: %w", err)
	}

	logger.Info("modules montés",
		slog.String("outbox", cfg.Modules[outbox.Name].Driver),
		slog.String("auth", cfg.Modules[auth.Name].Driver),
		slog.String("user_registration", orDefault(driver, userregistration.DriverMemory)),
	)
	return assembled{outbox: outboxMod, auth: authMod, users: users}, nil
}

// generateUserID produit un identifiant ordonné dans le temps.
//
// UUID v7 et non v4 : l'identifiant devient une clé primaire, et une clé
// aléatoire disperse les insertions sur tout l'index
// (rules/donnees-et-migrations.md §7).
//
// Le repli sur v4 en cas d'échec est délibéré : `NewV7` n'échoue que si
// l'entropie du système est indisponible, et refuser une inscription pour cette
// raison serait pire qu'un identifiant moins bien ordonné.
func generateUserID() domain.UserID {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.UserID(uuid.NewString())
	}
	return domain.UserID(id.String())
}

// probes déclare ce que /readyz vérifie.
//
// /healthz ne vérifie RIEN, volontairement : sinon un incident de base ferait
// redémarrer tous les conteneurs, transformant une panne partielle en
// indisponibilité totale. /readyz, lui, retire l'instance du service.
func probes(mods assembled) map[string]httpserver.Probe {
	return map[string]httpserver.Probe{
		// L'outbox est la dépendance dont la panne est SILENCIEUSE : si le
		// dépileur meurt, tout continue de répondre pendant que les événements
		// s'accumulent. Compter les messages en attente est le seul symptôme
		// observable, donc la sonde la plus utile du système.
		"outbox": func(ctx context.Context) error {
			if _, err := mods.outbox.PendingCount(ctx); err != nil {
				return fmt.Errorf("outbox injoignable: %w", err)
			}
			return nil
		},
	}
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// moduleCatalog assemble ce que la configuration a le droit de nommer.
//
// Le noyau apporte le sien ; ce binaire y ajoute celui de chaque module métier
// qu'il embarque. Aucun fichier du framework ne nomme un module métier — c'est
// très exactement la friction que l'ADR 014 supprime.
func moduleCatalog() (config.ModuleCatalog, error) {
	coreCatalog, err := core.Catalog()
	if err != nil {
		return nil, fmt.Errorf("catalogue du noyau: %w", err)
	}
	// Les deux binaires lisent le MÊME `config/modules.yaml` : l'ensemble des
	// modules déclarables est donc une propriété de l'application, pas du
	// binaire. Voir internal/modules/catalog.go — le dépileur refusait de
	// démarrer tant que ce n'était pas le cas.
	businessCatalog, err := modules.Catalog()
	if err != nil {
		return nil, fmt.Errorf("catalogue des modules métier: %w", err)
	}
	merged, err := config.MergeCatalogs(coreCatalog, businessCatalog)
	if err != nil {
		return nil, fmt.Errorf("fusion des catalogues: %w", err)
	}
	return merged, nil
}
