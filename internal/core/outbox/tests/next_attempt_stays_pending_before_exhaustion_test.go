package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptStaysPendingBeforeExhaustion(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 1}, 5, time.Second, fixedNow(), "raison")
	if got.Status != domain.StatusPending {
		t.Errorf("statut = %q, attendu pending", got.Status)
	}
}
