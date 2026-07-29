package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptBackoffIsExponentialAndBounded(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	base := time.Second

	cases := map[string]struct {
		attempts int
		want     time.Duration
	}{
		"first failure":  {attempts: 0, want: 2 * time.Second},
		"second failure": {attempts: 1, want: 4 * time.Second},
		"third":          {attempts: 2, want: 8 * time.Second},
		// The shift is bounded: without a bound, 1<<40 would overflow and produce
		// a NEGATIVE duration, hence a message replayed immediately in a loop.
		"shift bounded at 2^10": {attempts: 40, want: 1024 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := domain.NextAttempt(
				domain.Message{Attempts: tc.attempts},
				domain.RetryPolicy{MaxAttempts: 100, BaseBackoff: base},
				now, "reason",
			)
			if delay := got.AvailableAt.Sub(now); delay != tc.want {
				t.Errorf("backoff = %v, want %v", delay, tc.want)
			}
			if got.AvailableAt.Before(now) {
				t.Error("the next try is in the past: duration overflow")
			}
		})
	}
}
