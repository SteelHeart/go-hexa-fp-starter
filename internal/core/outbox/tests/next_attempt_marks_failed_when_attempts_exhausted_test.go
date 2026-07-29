package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptMarksFailedWhenAttemptsExhausted(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 4},
		domain.RetryPolicy{MaxAttempts: 5, BaseBackoff: time.Second},
		fixedNow(), "reason")
	if got.Status != domain.StatusFailed {
		t.Errorf("status = %q, want failed after the attempts are exhausted", got.Status)
	}

	// An abandoned message is NEVER deleted: it is the only trace of what has
	// not been published.
	if got.Reason == "" {
		t.Error("the reason for the abandonment must be kept")
	}
}
