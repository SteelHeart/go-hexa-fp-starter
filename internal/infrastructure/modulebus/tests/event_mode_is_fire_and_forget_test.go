package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestEventModeIsFireAndForget: the event goes out, and there is NO reply.
//
// # What this test locks down
//
// The `event` mode returns the zero value with `nil`. That is correct — there
// is no reply in an asynchronous send — and it is a trap: the same signature as
// the other two modes makes an absence of reply indistinguishable from a
// negative reply.
//
// The test therefore asserts it explicitly, so that this property is a readable
// DECISION and not a side effect. Configuration alone decides the mode:
// enabling it on a capability whose caller reads the result would make it read
// "refused" on every call, with no error at all.
//
// ⚠️ KNOWN NON-GUARANTEE, here because a test is the only place that gets
// re-read: the envelope produced carries NO AggregateID — the bus is generic,
// it does not know which entity the call concerns. Consequence on a Kafka
// relay: the partition key is empty, so no ordering is guaranteed between two
// events of the same entity, and they all land on the same partition. For a
// fire-and-forget capability that is acceptable; it is not for a capability
// whose ordering matters. To be settled before opening this mode up widely.
func TestEventModeIsFireAndForget(t *testing.T) {
	t.Parallel()

	var published []messaging.Envelope
	publisher := func(_ context.Context, env messaging.Envelope) error {
		published = append(published, env)
		return nil
	}

	var localCalls int
	call := resolve(t, interop("event", nil), publisher, localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-7"})
	if err != nil {
		t.Fatalf("posting the event failed: %v", err)
	}

	if got.Accepted {
		t.Errorf("reply = %+v, want the zero value: an asynchronous send has no reply", got)
	}
	if localCalls != 0 {
		t.Errorf("the local implementation was called %d times in event mode", localCalls)
	}
	if len(published) != 1 {
		t.Fatalf("%d event(s) published, want 1", len(published))
	}

	env := published[0]
	if env.Type != someEvent {
		t.Errorf("type published = %q, want %q", env.Type, someEvent)
	}
	if env.ID == "" {
		t.Error("event with no identifier — a consumer can no longer deduplicate")
	}
	if env.OccurredAt.IsZero() {
		t.Error("event with no timestamp")
	}
	var payload request
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("unreadable payload: %v", err)
	}
	if payload.Ref != "r-7" {
		t.Errorf("payload = %+v, the request did not cross over", payload)
	}
}

// TestEventModeSurfacesAPublicationFailure: a failed posting is not a success.
//
// # The defect this test catches
//
// This is the trickiest point of this mode. "Fire-and-forget" qualifies the
// absence of a REPLY, not the absence of a GUARANTEE: if publication fails, the
// call must fail. Swallowing the error would let the caller believe it was
// served while nothing went out — and since the mode never returns a reply,
// there is no other signal. The capability would be dead in silence,
// permanently.
func TestEventModeSurfacesAPublicationFailure(t *testing.T) {
	t.Parallel()

	refused := errors.New("relay unavailable")
	publisher := func(context.Context, messaging.Envelope) error { return refused }

	var localCalls int
	call := resolve(t, interop("event", nil), publisher, localCaller(&localCalls))

	_, err := call(context.Background(), request{Ref: "r-7"})

	if err == nil {
		t.Fatal("a failed publication returned nil — the capability would be dead in silence")
	}
	if !errors.Is(err, refused) {
		t.Errorf("the original cause is lost: %v", err)
	}
}
