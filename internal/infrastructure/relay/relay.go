// Package relay plugs the outbox dispatcher onto the event transport.
//
// # Why this package exists, and why HERE
//
// The outbox is a CORE module: it must import no infrastructure, on pain of no
// longer being extractable into an independent Go module (ADR 012). The
// transport, for its part, knows nothing of the outbox — and rightly so: it
// publishes envelopes, without knowing where they come from.
//
// Someone must nonetheless connect them. This package is that someone: it
// consumes both and is consumed only by a composition root. The dependency
// therefore goes from the infrastructure towards the core, never the reverse.
//
// It lives outside `cmd/` for a precise reason: a field-to-field mapping looks
// trivial and is not. Forgetting `Payload` would publish empty envelopes;
// forgetting `TraceParent` would cut the trace between producer and consumer.
// In both cases the dispatcher would report `published`, the message would be
// marked as handled, and nothing would signal the loss. This code must
// therefore be testable — and code inside `main` is only half testable.
package relay

import (
	"context"
	"fmt"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	outboxports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// FromOutbox builds the handler that the dispatcher calls for each claimed
// message.
//
// The publication error goes back up AS IS. That is essential: the exponential
// backoff and the abandonment after N attempts are decided by a pure and tested
// policy, in `outbox/application`. Swallowing the error here would mark the
// message as handled although nothing has left — hence definitively lost,
// without a trace. And catching it to decide by itself would duplicate the
// policy, with the certainty that the two will diverge.
func FromOutbox(publish messaging.Publisher) outboxports.Handler {
	return func(ctx context.Context, msg outboxdomain.Message) error {
		if err := publish(ctx, envelopeOf(msg)); err != nil {
			return fmt.Errorf("publication of %s: %w", msg.Type, err)
		}
		return nil
	}
}

// envelopeOf translates a persisted message into a transport envelope.
//
// `OccurredAt` carries the CREATION date of the message, not that of its
// publication: a consumer must be able to order facts according to the moment
// they occurred, and not according to the moment the dispatcher took them out.
// After a dispatcher breakdown, the two differ by several hours.
func envelopeOf(msg outboxdomain.Message) messaging.Envelope {
	return messaging.Envelope{
		ID:          msg.ID.String(),
		Type:        msg.Type,
		AggregateID: msg.AggregateID,
		Payload:     msg.Payload,
		TraceParent: msg.TraceParent,
		OccurredAt:  msg.CreatedAt,
		Headers:     msg.Headers,
	}
}
