package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryGivesUpAndKeepsTheLastCause : le réessai est BORNÉ, et il rend compte.
//
// # Les deux défauts que ce test attrape
//
//  1. Un réessai non borné bloquerait le dépileur sur un seul message. Le
//     recul de l'outbox — minutes, survivant à un redémarrage — est la vraie
//     politique de reprise ; celui-ci n'absorbe qu'une coupure de l'ordre de la
//     seconde. Les confondre transforme une panne de broker en arrêt du
//     dépilage : plus AUCUN événement ne sort, pas seulement celui qui échoue.
//  2. Une erreur remplacée par un message générique. Le dépileur journalise ce
//     qu'on lui rend ; s'il reçoit « publication échouée » sans la cause, la
//     panne se diagnostique en devinant.
func TestRetryGivesUpAndKeepsTheLastCause(t *testing.T) {
	t.Parallel()

	const attempts = 3
	cause := errors.New("broker injoignable")

	var calls int
	always := func(context.Context, messaging.Envelope) error {
		calls++
		return cause
	}

	publish := messaging.WithRetry(always, attempts, time.Millisecond)
	err := publish(context.Background(), envelope("user.registered.v1"))

	if err == nil {
		t.Fatal("un publieur toujours en échec a rendu nil — l'événement serait marqué publié")
	}
	if !errors.Is(err, cause) {
		t.Errorf("la cause d'origine est perdue: %v", err)
	}
	if calls != attempts {
		t.Errorf("publieur appelé %d fois, attendu exactement %d — "+
			"le réessai doit rester borné", calls, attempts)
	}
}
