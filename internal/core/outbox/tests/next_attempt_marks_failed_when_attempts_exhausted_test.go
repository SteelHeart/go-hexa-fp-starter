package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestNextAttemptMarksFailedWhenAttemptsExhausted(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 4}, 5, time.Second, fixedNow(), "raison")
	if got.Status != domain.StatusFailed {
		t.Errorf("statut = %q, attendu failed après épuisement des tentatives", got.Status)
	}

	// Un message abandonné n'est JAMAIS supprimé : c'est la seule trace de ce
	// qui n'a pas été publié.
	if got.Reason == "" {
		t.Error("la raison de l'abandon doit être conservée")
	}
}
