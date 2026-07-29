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

// errIdempotencyRequired refuses to consume without a replay guard.
//
// # Why this is a refusal and not a warning
//
// Every transport here is "at least once": the same envelope arrives twice as
// soon as an acknowledgement is lost, which is commonplace. Consuming without
// idempotency would therefore send the welcome email twice — and the day a
// consumer charges a card, it will be the charge.
//
// A warning at start-up would be read once then never. The refusal names what
// has to change, and it is repairable in one line of configuration.
var errIdempotencyRequired = errors.New(
	"consuming events without idempotency would replay the effects — enable modules.idempotency")

// subscribe wires the event consumers onto the relay.
//
// # Three outcomes, and each one is a decision
//
//	notification disabled  → no subscription, and we SAY so at start-up
//	idempotency disabled   → REFUSAL to start
//	both enabled           → subscription guarded against replay
//
// The first case does not fail the start-up: a module that is off is a
// configured state, not a failure. But silence would be worse than inaction —
// a dispatcher publishing to nobody looks in every respect like a dispatcher
// that works. The log therefore names the consequence.
func subscribe(cfg config.Config, broker messaging.Broker, logger *slog.Logger) error {
	notifCfg := cfg.Modules[notification.Name]
	if !notifCfg.Enabled {
		logger.Warn("no subscription to user.registered.v1 — modules.notification is disabled",
			slog.String("consequence", "the events are published and nobody reacts to them"),
		)
		return nil
	}

	idemCfg := cfg.Modules[idempotency.Name]
	if !idemCfg.Enabled {
		return errIdempotencyRequired
	}

	notif, err := notification.New(notifCfg, notification.Deps{Logger: logger})
	if err != nil {
		return fmt.Errorf("notification module: %w", err)
	}
	idem, err := idempotency.New(idemCfg, idempotency.Deps{Now: time.Now})
	if err != nil {
		return fmt.Errorf("idempotency module: %w", err)
	}

	guard := subscriber.Guard{Reserve: idem.Reserve, Complete: idem.Complete, Release: idem.Release}

	// The ORDER of the decorators is the decision: the trace is restored
	// BEFORE the idempotency guard, so that the reservation itself belongs to
	// the producer's trace. The other way round, a refused replay would appear
	// in no trace at all — and that is exactly the case we are trying to
	// explain.
	broker.Consume.Subscribe(
		contract.EventUserRegisteredV1,
		subscriber.WithTrace(subscriber.Once(guard, welcome(notif))),
	)

	logger.Info("subscription mounted",
		slog.String("evenement", contract.EventUserRegisteredV1),
		slog.String("notification", notifCfg.Driver),
		slog.String("idempotence", idemCfg.Driver),
	)
	return nil
}
