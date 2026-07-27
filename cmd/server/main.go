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
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/outboxpub"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"

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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	logger := telemetry.NewLogger(cfg)
	logger.InfoContext(ctx, "démarrage",
		slog.String("service", cfg.App.Name),
		slog.String("env", string(cfg.App.Env)),
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", buildDate),
	)

	modules, err := compose(cfg, logger)
	if err != nil {
		return err
	}

	router := httpserver.NewRouter(cfg, logger, probes(modules))
	userhttp.Mount(router.API, modules.users)
	userhttp.MountAvailability(router.API, modules.users)

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

	// Le pilote vient de la configuration si elle le nomme, du défaut sinon.
	//
	// ⚠️ Point de conception OUVERT : `config/modules.yaml` ne valide aujourd'hui
	// que les modules NOYAU. Un module métier ne peut donc pas encore y déclarer
	// ses pilotes, et son `module.go` les valide lui-même. Faire déclarer ses
	// pilotes par une application sans qu'elle modifie un fichier du socle est un
	// vrai sujet de framework, pas un détail — il est ouvert, pas oublié.
	driver := cfg.Modules[userregistration.Name].Driver

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
		slog.String("user_registration", orDefault(driver, userregistration.DriverMemory)),
	)
	return assembled{outbox: outboxMod, users: users}, nil
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
