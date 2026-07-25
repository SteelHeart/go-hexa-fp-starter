package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestFailedPublicationIsRescheduled : un courtier injoignable n'est pas une perte.
//
// Le message reste `pending`, son compteur monte, et sa date de disponibilité est
// repoussée. Marquer `done` un message non publié serait la perte silencieuse que
// tout le patron outbox existe pour empêcher.
func TestFailedPublicationIsRescheduled(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	panne := errors.New("courtier injoignable")
	handle := func(context.Context, domain.Message) error { return panne }

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	if len(observed.done) != 0 {
		t.Error("un message non publié ne doit JAMAIS être marqué comme traité")
	}

	attempt := observed.lastAttempt(t)
	if attempt.Status != domain.StatusPending {
		t.Errorf("statut = %q, attendu %q : le message doit rester rejouable",
			attempt.Status, domain.StatusPending)
	}
	if attempt.Attempts != 1 {
		t.Errorf("tentatives = %d, attendu 1", attempt.Attempts)
	}
	if !attempt.AvailableAt.After(fixedNow()) {
		t.Error("le prochain essai doit être repoussé dans le futur")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventRetryScheduled {
		t.Errorf("événement = %q, attendu %q", got, domain.EventRetryScheduled)
	}
}
