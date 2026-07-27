package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestExhaustedMessageStopsBeingReplayed : après N essais, on arrête.
//
// Un message empoisonné — charge illisible par le consommateur, destinataire
// disparu — ne se publiera jamais. Le rejouer éternellement noierait les journaux
// et occuperait chaque lot, retardant les messages sains derrière lui.
//
// Il passe donc en `failed` et n'est plus repris. Il n'est PAS supprimé : c'est la
// seule trace de ce qui n'a pas été publié, et l'événement mérite une intervention
// humaine — d'où le compte rendu distinct.
func TestExhaustedMessageStopsBeingReplayed(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	handle := func(context.Context, domain.Message) error {
		return errors.New("charge illisible")
	}

	// La politique de test autorise trois tentatives au total.
	policy := testPolicy()
	// Le message a déjà échoué deux fois : cet essai est le troisième.
	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 2)), handle), policy)

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	attempt := observed.lastAttempt(t)
	if attempt.Attempts != policy.Retry.MaxAttempts {
		t.Errorf("tentatives = %d, attendu %d", attempt.Attempts, policy.Retry.MaxAttempts)
	}
	if attempt.Status != domain.StatusFailed {
		t.Errorf("statut = %q, attendu %q", attempt.Status, domain.StatusFailed)
	}
	if attempt.Reason == "" {
		t.Error("la cause doit être conservée : sans elle, la trace est inexploitable")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventExhausted {
		t.Errorf("événement = %q, attendu %q", got, domain.EventExhausted)
	}
}
