// Commande worker dépile l'outbox et publie vers le relais de messagerie.
//
// # La moitié asynchrone de la chaîne
//
// Un cas d'usage écrit son événement dans l'outbox, DANS sa transaction métier :
// c'est ce qui garantit qu'aucun événement n'est perdu ni fantôme (ADR 006).
// Sans ce binaire, l'événement est durable et n'est jamais publié — la chaîne
// s'arrête à mi-parcours, et rien ne le signale côté serveur.
//
// # Ce binaire REFUSE de démarrer sur le pilote `memory`, et c'est voulu
//
// Le pilote `memory` de l'outbox vit dans le processus. Un worker séparé
// dépilerait donc SON magasin, vide, pendant que les événements du serveur
// resteraient dans la mémoire du serveur. Il tournerait sans rien publier, sans
// aucune erreur, et le défaut ne se verrait que chez le consommateur qui
// n'aurait jamais rien reçu.
//
// Un composant silencieusement inerte est le pire défaut possible. Le worker
// refuse donc explicitement, en disant quoi changer.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	outboxapp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/ressources"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
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
		fmt.Fprintf(os.Stderr, "démarrage impossible: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	arret, err := demarrerObservabilite(ctx, cfg)
	if err != nil {
		return err
	}
	defer arreterObservabilite(ctx, arret, logger)

	ctx, couper := context.WithCancel(ctx)
	defer couper()
	servirMetriques(ctx, cfg, logger, couper)

	logger.InfoContext(ctx, "démarrage du dépileur",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", buildDate),
	)

	return depiler(ctx, cfg, catalog, logger)
}

// depiler ouvre les connexions, monte le dépileur et le tient jusqu'à l'arrêt.
//
// Extraite de `run` parce que celle-ci dépassait le seuil de lignes d'`arch-go`
// en y branchant l'ouverture du pool. Le garde a attrapé la régression avant la
// relecture — c'est exactement ce qu'on lui demande, et c'est la deuxième fois.
//
// La coupure n'est pas arbitraire : `run` porte l'AMORÇAGE — signaux, catalogue,
// configuration, journal, observabilité — et `depiler` porte le TRAVAIL.
func depiler(
	ctx context.Context, cfg config.Config, catalog config.ModuleCatalog, logger *slog.Logger,
) error {
	// Les connexions AVANT les modules : un pilote `postgres` reçoit un pool ou
	// refuse le démarrage, il ne tombe jamais en panne à la première requête.
	//
	// Rien n'est ouvert si aucun module activé n'en réclame — c'est la promesse
	// de l'ADR 012, et corriger #103 ne devait pas la coûter.
	conn, err := ressources.Open(ctx, cfg, catalog)
	if err != nil {
		return fmt.Errorf("ouverture des connexions: %w", err)
	}
	defer conn.Close()

	w, err := compose(cfg, conn, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := w.closeBroker(); err != nil {
			logger.Error("fermeture du relais en échec", slog.Any("error", err))
		}
	}()

	logger.InfoContext(ctx, "dépilage en cours",
		slog.String("outbox_driver", cfg.Modules[outbox.Name].Driver),
		slog.String("relais", cfg.Messaging.Driver),
	)

	if err := w.boucler(ctx); err != nil {
		return err
	}
	logger.Info("dépileur arrêté proprement")
	return nil
}

// worker est le dépileur monté, avec ce qu'il faut pour le libérer.
//
// Type plutôt que trois retours : `compose` rendait
// `(*Dispatcher, Closer, error)`, et la règle d'architecture l'a refusé. Elle a
// raison, et c'est la CINQUIÈME fois que ce dépôt paie la même leçon — après
// `election`, `decodedHash`, `RetryPolicy` et `messaging.Broker`. Plus de deux
// valeurs de retour signale toujours un type qui manque.
type worker struct {
	dispatch    *outboxapp.Dispatcher
	consume     messaging.Consumer
	closeBroker messaging.Closer
}

// compose monte le relais, l'outbox et le dépileur.
//
// Rend le libérateur du courtier plutôt que de le fermer lui-même : la durée de
// vie du courtier est celle du processus, pas celle de cette fonction.
func compose(cfg config.Config, conn ressources.Connexions, logger *slog.Logger) (worker, error) {
	outboxCfg := cfg.Modules[outbox.Name]
	if !outboxCfg.Enabled {
		return worker{}, fmt.Errorf("%w: modules.outbox.enabled est false", outbox.ErrDisabled)
	}
	// Le serveur, lui, n'a pas cette exigence : il ÉCRIT dans l'outbox et n'a donc
	// aucune raison de réclamer un pilote partagé. Seul un dépileur SÉPARÉ en a
	// besoin, et c'est pourquoi le refus vit ici et non dans le module.
	if err := outbox.RequireSharedDriver(outboxCfg.Driver); err != nil {
		return worker{}, fmt.Errorf("configuration du dépileur: %w", err)
	}

	broker, err := messaging.New(cfg.Messaging, logger)
	if err != nil {
		return worker{}, fmt.Errorf("relais de messagerie: %w", err)
	}

	// Les abonnements AVANT le dépileur : `Subscribe` doit être appelé avant
	// `Run`, et un dépileur qui publierait pendant que les abonnements se
	// montent perdrait les premières enveloppes sans le dire.
	//
	// Variable NOMMÉE plutôt que `err` réutilisé : `govet shadow` et
	// `gocritic sloppyReassign` se contredisent sur cette ligne — l'un veut `=`,
	// l'autre `:=`. Un nom distinct sort du conflit sans faire taire ni l'un ni
	// l'autre, et se relit mieux.
	if erreurAbonnement := abonner(cfg, broker, logger); erreurAbonnement != nil {
		return worker{}, erreurAbonnement
	}

	// Le pool peut être nil, légitimement : il l'est dès qu'aucun module activé
	// ne réclame de base. Un pilote qui en a besoin refuse alors le démarrage en
	// le disant — c'est le garde qui existait déjà et que personne n'atteignait.
	mod, err := outbox.New(outboxCfg, outbox.Deps{Pool: conn.Pool, Now: time.Now})
	if err != nil {
		return worker{}, fmt.Errorf("module outbox: %w", err)
	}

	dispatch, err := outbox.NewDispatcher(mod, outboxCfg, outbox.DispatcherDeps{
		Handle: relay.FromOutbox(broker.Publish),
		Logger: logger,
		Now:    time.Now,
	})
	if err != nil {
		return worker{}, fmt.Errorf("dépileur de l'outbox: %w", err)
	}
	return worker{dispatch: dispatch, consume: broker.Consume, closeBroker: broker.Close}, nil
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
