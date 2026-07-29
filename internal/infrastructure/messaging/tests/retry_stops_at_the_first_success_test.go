package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryStopsAtTheFirstSuccess: a successful retry does not republish.
//
// # The defect this test catches
//
// A loop that did not exit on success would publish the same event several
// times. Every transport here is "at least once", so the duplicate is not a
// correctness fault — but it is a COST fault: each duplicate forces a consumer
// to look up its idempotency record, and the starter would produce them for
// free, permanently.
func TestRetryStopsAtTheFirstSuccess(t *testing.T) {
	t.Parallel()

	var calls int
	flaky := func(context.Context, messaging.Envelope) error {
		calls++
		if calls < 2 {
			return errors.New("network outage")
		}
		return nil
	}

	publish := messaging.WithRetry(flaky, 5, time.Millisecond)

	if err := publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("the publication was to succeed on the 2nd attempt, returned: %v", err)
	}
	if calls != 2 {
		t.Errorf("publisher called %d times, want 2 — every extra attempt is a duplicate", calls)
	}
}
