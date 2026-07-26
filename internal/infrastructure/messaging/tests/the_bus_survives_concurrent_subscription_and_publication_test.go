package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheBusSurvivesConcurrentSubscriptionAndPublication : abonner pendant qu'on
// publie ne corrompt pas la table.
//
// # Le défaut que ce test attrape
//
// Publier en tenant le verrou en LECTURE pendant qu'un consommateur s'abonne en
// ÉCRITURE est le scénario réel du démarrage : les modules se montent pendant que
// le dépileur tourne déjà. Une lecture de la carte sans copie — `handlers :=
// b.handlers[env.Type]` au lieu d'une copie — laisse la tranche partagée avec un
// `append` concurrent, et Go n'a alors AUCUNE garantie.
//
// ⚠️ Ce test n'a toute sa force que sous `-race`, exécuté en CI (F005 : `-race`
// exige CGO, absent de la machine de référence). Sans lui, il exerce le verrou
// mais ne prouve pas l'absence de course.
func TestTheBusSurvivesConcurrentSubscriptionAndPublication(t *testing.T) {
	t.Parallel()

	const rounds = 50

	bus := messaging.NewInproc(quietLogger())
	var delivered atomic.Int64

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for range rounds {
			bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
				delivered.Add(1)
				return nil
			})
		}
	}()
	go func() {
		defer wg.Done()
		for range rounds {
			if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
				t.Errorf("publication concurrente en échec: %v", err)
			}
		}
	}()
	wg.Wait()

	// Le NOMBRE de livraisons dépend de l'entrelacement, donc on ne l'affirme
	// pas : un test qui figerait ce nombre serait instable pour une bonne raison.
	// Ce qui est affirmé, c'est qu'une fois tout le monde abonné, tout le monde
	// reçoit.
	before := delivered.Load()
	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("publication finale en échec: %v", err)
	}
	if got := delivered.Load() - before; got != rounds {
		t.Errorf("%d consommateurs servis après la course, attendu %d — "+
			"un abonnement a été perdu", got, rounds)
	}
}
