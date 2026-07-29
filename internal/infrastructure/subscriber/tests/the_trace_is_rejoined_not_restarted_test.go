package tests

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// TestTheTraceIsRejoinedNotRestarted is the witness of the third criterion of
// #9.
//
// # What the oversight produces, and why nobody notices it
//
// Everything keeps working. The events leave, the effects take place, the tests
// pass. Only the asynchronous half of a request becomes an ORPHAN trace: the
// registration has its own, the email sending has its own, and nothing links
// them.
//
// On the day a sending fails, one cannot go back up to the registration that
// caused it — and that is precisely the day one needs to. It is the same fault
// that `relay` documents in the other direction: forgetting `TraceParent` at
// dispatching time cuts the chain on the producer side. Both halves must be
// wired for either one to serve any purpose.
//
// The test compares the trace identifier RECEIVED by the handler with the one
// written in the envelope. Checking only that a context "has a trace" would
// pass with a brand new trace — that is to say, with the defect.
func TestTheTraceIsRejoinedNotRestarted(t *testing.T) {
	// No `t.Parallel()`: this test installs the GLOBAL propagator, the one
	// `telemetry.Setup` configures in production. Building a local one would
	// exercise a propagator the producer does not use.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	const (
		traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		spanID  = "00f067aa0ba902b7"
	)
	parent := "00-" + traceID + "-" + spanID + "-01"

	effect := &counter{}
	env := envelope("evt-trace")
	env.TraceParent = parent

	if err := subscriber.WithTrace(effect.handler())(context.Background(), env); err != nil {
		t.Fatalf("the handler must not fail: %v", err)
	}

	span := trace.SpanContextFromContext(effect.lastContext(t))
	if !span.IsValid() {
		t.Fatal("no trace context restored: the asynchronous half would be an orphan")
	}
	if span.TraceID().String() != traceID {
		t.Fatalf("trace RESTARTED instead of being rejoined: %s instead of %s",
			span.TraceID(), traceID)
	}
	if span.SpanID().String() != spanID {
		t.Fatalf("parent span lost: %s instead of %s", span.SpanID(), spanID)
	}
}

// TestAnEnvelopeWithoutTraceIsStillDelivered guards delivery before the trace.
//
// An envelope without a `traceparent` is NOT an error: the original context
// leaves intact and a new segment begins. Refusing it would lose an event over
// an observability defect — sacrificing what one measures to the way one
// measures it.
//
// The case is not theoretical: every producer written before telemetry was
// wired in (#13) left the field empty.
func TestAnEnvelopeWithoutTraceIsStillDelivered(t *testing.T) {
	t.Parallel()

	effect := &counter{}
	if err := subscriber.WithTrace(effect.handler())(context.Background(), envelope("evt-without-trace")); err != nil {
		t.Fatalf("an envelope without a trace must be delivered: %v", err)
	}
	if effect.total() != 1 {
		t.Fatalf("the effect was to take place, got %d calls", effect.total())
	}
}

// TestAMalformedTraceParentDoesNotDropTheEnvelope guards delivery too.
//
// An unreadable header must not cost an event. The propagator ignores what it
// does not understand, and the handler is called all the same — the delivery of
// the message counts more than the continuity of its trace.
func TestAMalformedTraceParentDoesNotDropTheEnvelope(t *testing.T) {
	t.Parallel()

	for _, parent := range []string{"not-a-traceparent", "00-", "99-xxx-yyy-zz"} {
		effect := &counter{}
		env := envelope("evt-broken-trace")
		env.TraceParent = parent

		if err := subscriber.WithTrace(effect.handler())(context.Background(), env); err != nil {
			t.Errorf("traceparent %q: the envelope must be delivered, got %v", parent, err)
		}
		if effect.total() != 1 {
			t.Errorf("traceparent %q: the effect was to take place", parent)
		}
	}
}
