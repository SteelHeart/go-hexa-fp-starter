package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryStopsAtTheFirstSuccess : un réessai réussi ne republie pas.
//
// # Le défaut que ce test attrape
//
// Une boucle qui ne sortirait pas au succès publierait le même événement
// plusieurs fois. Tous les transports d'ici sont « au moins une fois », donc le
// doublon n'est pas une faute de correction — mais c'est une faute de COÛT :
// chaque doublon force un consommateur à retrouver son enregistrement
// d'idempotence, et le socle en produirait gratuitement, en permanence.
func TestRetryStopsAtTheFirstSuccess(t *testing.T) {
	t.Parallel()

	var calls int
	flaky := func(context.Context, messaging.Envelope) error {
		calls++
		if calls < 2 {
			return errors.New("coupure réseau")
		}
		return nil
	}

	publish := messaging.WithRetry(flaky, 5, time.Millisecond)

	if err := publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("la publication devait réussir au 2e essai, rendu: %v", err)
	}
	if calls != 2 {
		t.Errorf("publieur appelé %d fois, attendu 2 — chaque essai en trop est un doublon", calls)
	}
}
