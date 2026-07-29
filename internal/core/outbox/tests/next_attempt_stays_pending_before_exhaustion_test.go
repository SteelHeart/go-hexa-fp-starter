package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptStaysPendingBeforeExhaustion(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 1},
		domain.RetryPolicy{MaxAttempts: 5, BaseBackoff: time.Second},
		fixedNow(), "reason")
	if got.Status != domain.StatusPending {
		t.Errorf("status = %q, want pending", got.Status)
	}
}
