// Package outboxpub wires the module's PublishEvent port onto the core outbox.
//
// # Why this package exists
//
// The use case must be able to publish an event without knowing the outbox, and
// the outbox must remain a core module with no notion of business whatsoever.
// This adapter is the only point where the two meet — which is exactly the role
// of a secondary adapter.
//
// It does three things, and nothing else:
//
//  1. serialises the payload as JSON (the core treats it as opaque);
//  2. translates the core's TECHNICAL error into a DOMAIN error;
//  3. logs nothing — the cause is attached to the error, which travels up.
//
// The translation is not a formality: without it, a driver error would travel up
// to the HTTP surface, which would return it as it is to the caller. That is an
// internal structure leak (rules/donnees-et-migrations.md §2).
package outboxpub

import (
	"context"
	"encoding/json"
	"errors"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	outboxports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// New builds the module's publication port.
//
// `traceParent` is read from the context by the caller upstream; it is not
// rebuilt here so that this adapter stays free of any dependency on telemetry.
func New(enqueue outboxports.Enqueue) ports.PublishEvent {
	return func(
		ctx context.Context,
		eventType string,
		aggregateID string,
		payload any,
	) result.Result[domain.Ack, domain.Error] {
		raw, err := json.Marshal(payload)
		if err != nil {
			// A non-serialisable event is a PROGRAMMING defect, not a
			// breakdown: it will not heal on retry. Hence CodeInternal, which
			// produces a 500 — and not CodeUnavailable, which would invite the
			// client to retry indefinitely a request that will always fail.
			return failure(domain.CodeInternal,
				"l'événement d'inscription n'a pas pu être préparé", err)
		}

		msg := outboxdomain.NewMessage{
			Type:        eventType,
			AggregateID: aggregateID,
			Payload:     raw,
		}

		if _, err := enqueue(ctx, msg); err != nil {
			return failure(translate(err),
				"l'inscription n'a pas pu être enregistrée", err)
		}
		return result.Ok[domain.Ack, domain.Error](domain.Ack{})
	}
}

// translate picks the domain code matching a technical breakdown.
//
// A cancelled context — client gone, deadline exceeded — is TRANSIENT: it
// amounts to CodeUnavailable, therefore 503, therefore retryable. Confusing it
// with CodeInternal would raise a breakdown alert every time a user closes their
// tab too early, and that kind of alert ends up being ignored.
func translate(err error) domain.ErrorCode {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.CodeUnavailable
	}
	return domain.CodeInternal
}

// failure builds the error while attaching the cause.
//
// The cause is carried by the error, never logged here: an adapter that logs AND
// returns produces two traces for one incident, and the second one arrives
// without a correlation identifier.
func failure(code domain.ErrorCode, message string, cause error) result.Result[domain.Ack, domain.Error] {
	return result.Err[domain.Ack, domain.Error](
		domain.NewError(code, message).WithCause(cause),
	)
}
