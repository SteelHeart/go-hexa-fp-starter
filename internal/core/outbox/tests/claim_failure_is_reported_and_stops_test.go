package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// TestClaimFailureIsReportedAndStops : un magasin injoignable arrête le tour sans
// rien perdre. Aucun message n'a été réservé, donc aucun n'est en danger.
func TestClaimFailureIsReportedAndStops(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	panne := errors.New("base injoignable")
	failingClaim := func(context.Context, int) ([]domain.Message, error) { return nil, panne }

	var handle ports.Handler = func(context.Context, domain.Message) error {
		t.Error("aucun message ne doit être publié quand la réservation échoue")
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, failingClaim, handle), testPolicy())

	count, err := dispatcher.DrainOnce(context.Background())
	if err == nil {
		t.Fatal("une réservation en échec doit remonter une erreur")
	}
	if count != 0 {
		t.Errorf("messages traités = %d, attendu 0", count)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventClaimFailed {
		t.Errorf("événement = %q, attendu %q", got, domain.EventClaimFailed)
	}
}
