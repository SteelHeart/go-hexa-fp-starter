package tests

import (
	"context"
	"errors"
	"testing"

	idempotencydomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// errProvider simulates a breakdown of the recipient of the effect.
var errProvider = errors.New("provider unreachable")

// TestAFailedHandlerIsReplayableAtOnce guards the release of the key.
//
// # What the oversight produces
//
// The key stays reserved. The envelope therefore becomes UNHANDLEABLE until it
// expires — twenty-four hours with the shipped configuration — and every replay
// runs into "already in flight" without anything explaining why.
//
// The remedy would then be worse than the disease: someone would shorten the
// TTL to "unblock", and would reopen the replay window that this module exists
// to close.
//
// The test observes the property that counts: after a failure, the next
// delivery RUNS — it is neither refused, nor taken for a replay.
func TestAFailedHandlerIsReplayableAtOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	guard := newGuard(t)
	env := envelope("evt-failure")

	failed := failing(errProvider)
	if err := subscriber.Once(guard, failed.handler())(ctx, env); !errors.Is(err, errProvider) {
		t.Fatalf("the handler failure must go back up INTACT, got %v", err)
	}
	if failed.total() != 1 {
		t.Fatalf("the handler was to be called once, got %d", failed.total())
	}

	// Same key, same envelope, on the SAME guard: the second delivery must run,
	// not be taken for a replay.
	succeeded := &counter{}
	if err := subscriber.Once(guard, succeeded.handler())(ctx, env); err != nil {
		t.Fatalf("the delivery after a failure must succeed: %v", err)
	}
	if succeeded.total() != 1 {
		t.Fatal("the delivery after a failure did not run: the key stayed locked")
	}
}

// TestAnEnvelopeWithoutIDIsRefusedNotSilentlyLetThrough refuses noisily.
//
// The identifier IS the idempotency key. Without it, no replay can be
// recognised — and a decorator that let it through would offer a SILENTLY
// absent guarantee, which is worse than no guard at all: one stops being wary.
func TestAnEnvelopeWithoutIDIsRefusedNotSilentlyLetThrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effect := &counter{}
	handler := subscriber.Once(newGuard(t), effect.handler())

	env := envelope("")
	if err := handler(ctx, env); !errors.Is(err, subscriber.ErrMissingID) {
		t.Fatalf("want ErrMissingID, got %v", err)
	}
	if effect.total() != 0 {
		t.Fatal("an envelope without an identifier must produce no effect")
	}
}

// TestTheHandlerErrorSurvivesTheDecorator guards the CAUSE of the failure.
//
// The handler error must go back up recognisable by `errors.Is`. Replacing it
// with a decorator error would lose the CAUSE — and the dispatcher, upstream,
// decides to replay or to give up on that cause.
//
// The test also checks that a release error does not overwrite it: the two are
// joined, because a key left locked is a distinct incident which is hard to
// diagnose when its message has disappeared.
func TestTheHandlerErrorSurvivesTheDecorator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errRelease := errors.New("store unreachable")

	guard := newGuard(t)
	guard.Release = func(context.Context, idempotencydomain.Key) error { return errRelease }

	failed := failing(errProvider)
	err := subscriber.Once(guard, failed.handler())(ctx, envelope("evt-double-failure"))

	if !errors.Is(err, errProvider) {
		t.Errorf("the handler cause must survive: %v", err)
	}
	if !errors.Is(err, errRelease) {
		t.Errorf("the release failure must be joined, not overwritten: %v", err)
	}
}
