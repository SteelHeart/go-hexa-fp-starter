package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestPublishedMessageIsMarkedDone : le chemin nominal, et le seul où la chaîne
// asynchrone est réellement bouclée.
//
// Publier sans marquer republierait le message au tour suivant, indéfiniment : le
// consommateur recevrait le même événement toutes les deux secondes, pour toujours.
func TestPublishedMessageIsMarkedDone(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	var delivered []domain.MessageID
	handle := func(_ context.Context, msg domain.Message) error {
		delivered = append(delivered, msg.ID)
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	count, err := dispatcher.DrainOnce(context.Background())
	if err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}
	if count != 1 {
		t.Errorf("messages traités = %d, attendu 1", count)
	}
	if len(delivered) != 1 || delivered[0] != "m-1" {
		t.Errorf("messages publiés = %v, attendu [m-1]", delivered)
	}
	if len(observed.done) != 1 || observed.done[0] != "m-1" {
		t.Errorf("messages marqués = %v, attendu [m-1]", observed.done)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventPublished {
		t.Errorf("événement = %q, attendu %q", got, domain.EventPublished)
	}
}
