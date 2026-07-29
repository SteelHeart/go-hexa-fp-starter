package subscriber

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// traceParentHeader is the W3C header the envelope carries.
const traceParentHeader = "traceparent"

// WithTrace takes back the trace context deposited by the producer.
//
// # What the oversight produces, and why nobody notices it
//
// Everything keeps working. The events leave, the effects take place, the tests
// pass. Only, the asynchronous half of a request appears as an ORPHAN trace:
// the registration has its trace, the email sending has its own, and nothing
// links them. On the day a sending fails, one cannot go back up to the
// registration that caused it — and that is precisely the day one needs to.
//
// It is the same defect that `relay` documents in the other direction:
// forgetting `TraceParent` at dispatching time cuts the chain on the producer
// side. Both halves must be wired for either one of them to serve any purpose.
//
// # An envelope WITHOUT a traceparent is not an error
//
// The original context then leaves intact, and a new segment begins. Refusing
// the envelope would lose an event over an observability defect — sacrificing
// what one measures to the way one measures it.
func WithTrace(handler messaging.Handler) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		return handler(restore(ctx, env), env)
	}
}

// restore extracts the trace context from the envelope.
//
// It goes through the GLOBAL propagator rather than a local instance: it is the
// one `telemetry.Setup` configures, hence the same one the producer used to
// write. Building one here would make the two halves diverge on the day the
// configuration changed, without any test seeing it.
func restore(ctx context.Context, env messaging.Envelope) context.Context {
	if env.TraceParent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{traceParentHeader: env.TraceParent}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
