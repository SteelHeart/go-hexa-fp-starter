package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestPanickingHandlerDoesNotKillTheDispatcher : un message empoisonné ne doit pas
// emporter le worker.
//
// Le publieur est du code d'appelant : un accès à une carte nil dans un
// consommateur suffit à paniquer. Sans filet, cette panique remonterait la
// goroutine et tuerait tout le processus — et, avec le pilote `memory`, laisserait
// le message réservé pour toujours, donc invisible ET jamais publié.
//
// La panique est donc traitée comme un échec ordinaire : politique de recul,
// abandon après N essais, et un compte rendu qui la distingue clairement d'une
// erreur retournée proprement.
func TestPanickingHandlerDoesNotKillTheDispatcher(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	handle := func(context.Context, domain.Message) error {
		var absent map[string]string
		// L'écriture dans une carte nil est VOULUE : c'est le moyen le plus court de
		// provoquer une vraie panique d'exécution, celle qu'un pilote défectueux
		// produirait. `panic()` littéral serait moins fidèle : le dépileur doit
		// survivre aux paniques qu'il n'a pas vues venir, pas seulement aux nôtres.
		absent["boum"] = "panique" //nolint:govet,staticcheck // panique provoquée exprès, c'est l'objet du test
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	// Ne doit pas paniquer : c'est l'assertion principale, et elle est implicite.
	count, err := dispatcher.DrainOnce(context.Background())
	if err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}
	if count != 1 {
		t.Errorf("messages traités = %d, attendu 1", count)
	}

	if len(observed.done) != 0 {
		t.Error("un message dont la publication a paniqué ne doit pas être marqué traité")
	}
	attempt := observed.lastAttempt(t)
	if attempt.Attempts != 1 {
		t.Errorf("tentatives = %d, attendu 1", attempt.Attempts)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventHandlerPanicked {
		t.Errorf("événement = %q, attendu %q", got, domain.EventHandlerPanicked)
	}
}
