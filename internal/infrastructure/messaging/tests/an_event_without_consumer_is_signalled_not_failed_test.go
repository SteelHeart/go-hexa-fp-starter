package tests

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnEventWithoutConsumerIsSignalledNotFailed: signalled, not failed.
//
// # The two opposite defects this test catches
//
// It is a balance, and both sides are expensive:
//
//  1. RETURNING AN ERROR would be wrong. The outbox dispatcher would interpret
//     "nobody is listening" as "publication failed": it would replay, with a
//     growing backoff, then ABANDON a perfectly valid event. Yet the absence of
//     a consumer is a NORMAL state of the starter — this very day,
//     `user.registered.v1` has no subscriber.
//  2. SAYING NOTHING would be wrong too. A badly mounted notification module, or
//     an event type renamed on one side only, produces exactly the same trace as
//     nominal operation: the email does not go out, and nothing explains it.
//
// The correct outcome is therefore: success, AND a warning that NAMES the type.
func TestAnEventWithoutConsumerIsSignalledNotFailed(t *testing.T) {
	t.Parallel()

	logger, logs := recordingLogger(slog.LevelWarn)
	bus := messaging.NewInproc(logger)

	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("an event without a consumer returned %v — "+
			"the dispatcher would take it for a failure and end up abandoning it", err)
	}

	trace := logs.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Errorf("the warning does not name the event type, trace=%q — "+
			"without the name, it is unusable", trace)
	}
}
