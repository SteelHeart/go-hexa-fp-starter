package main

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// errIdempotenceRequise refuse de consommer sans garde de rejeu.
//
// # Pourquoi c'est un refus et non un avertissement
//
// Tous les transports d'ici sont « au moins une fois » : la même enveloppe
// arrive deux fois dès qu'un accusé de réception se perd, ce qui est banal.
// Consommer sans idempotence enverrait donc le courriel de bienvenue en double —
// et le jour où un consommateur débitera une carte, ce sera le débit.
//
// Un avertissement au démarrage serait lu une fois puis jamais. Le refus nomme
// ce qu'il faut changer, et il est réparable en une ligne de configuration.
var errIdempotenceRequise = errors.New(
	"consommer des événements sans idempotence rejouerait les effets — activer modules.idempotency")

// abonner branche les consommateurs d'événements sur le relais.
//
// # Trois issues, et chacune est une décision
//
//	notification désactivée  → aucun abonnement, et on le DIT au démarrage
//	idempotence désactivée   → REFUS de démarrer
//	les deux actives         → abonnement gardé contre le rejeu
//
// Le premier cas ne fait pas échouer le démarrage : un module éteint est un état
// configuré, pas une panne. Mais le silence serait pire que l'inaction — un
// dépileur qui publie vers personne ressemble en tout point à un dépileur qui
// fonctionne. Le journal nomme donc la conséquence.
func abonner(cfg config.Config, broker messaging.Broker, logger *slog.Logger) error {
	notifCfg := cfg.Modules[notification.Name]
	if !notifCfg.Enabled {
		logger.Warn("aucun abonnement à user.registered.v1 — modules.notification est désactivé",
			slog.String("consequence", "les événements sont publiés et personne n'y réagit"),
		)
		return nil
	}

	idemCfg := cfg.Modules[idempotency.Name]
	if !idemCfg.Enabled {
		return errIdempotenceRequise
	}

	notif, err := notification.New(notifCfg, notification.Deps{Logger: logger})
	if err != nil {
		return fmt.Errorf("module notification: %w", err)
	}
	idem, err := idempotency.New(idemCfg, idempotency.Deps{Now: time.Now})
	if err != nil {
		return fmt.Errorf("module idempotency: %w", err)
	}

	guard := subscriber.Guard{Reserve: idem.Reserve, Complete: idem.Complete, Release: idem.Release}

	// L'ORDRE des décorateurs est la décision : la trace est restaurée AVANT le
	// garde d'idempotence, pour que la réservation elle-même appartienne à la
	// trace du producteur. Dans l'autre sens, un rejeu refusé n'apparaîtrait
	// dans aucune trace — et c'est exactement le cas qu'on cherche à expliquer.
	broker.Consume.Subscribe(
		contract.EventUserRegisteredV1,
		subscriber.WithTrace(subscriber.Once(guard, bienvenue(notif))),
	)

	logger.Info("abonnement monté",
		slog.String("evenement", contract.EventUserRegisteredV1),
		slog.String("notification", notifCfg.Driver),
		slog.String("idempotence", idemCfg.Driver),
	)
	return nil
}
