package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheInprocBusDeliversToEverySubscriber : plusieurs consommateurs, un événement.
//
// # Le défaut que ce test attrape
//
// Une table `map[string]Handler` au lieu de `map[string][]Handler` — c'est ce
// qu'écrivent les deux relais réseau — remplace silencieusement le consommateur
// précédent. Le second module qui s'abonne à `user.registered.v1` désabonne le
// premier, et personne ne le voit : chaque module a son test, chaque test passe
// seul.
//
// Le test vérifie AUSSI que l'événement n'atteint pas un autre type : un bus qui
// diffuse à tout le monde marche en développement, et fait exécuter la facturation
// sur un événement d'inscription en production.
func TestTheInprocBusDeliversToEverySubscriber(t *testing.T) {
	t.Parallel()

	bus := messaging.NewInproc(quietLogger())

	var first, second, other int
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		first++
		return nil
	})
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		second++
		return nil
	})
	bus.Subscribe("invoice.issued.v1", func(context.Context, messaging.Envelope) error {
		other++
		return nil
	})

	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("publication en échec: %v", err)
	}

	if first != 1 || second != 1 {
		t.Errorf("consommateurs appelés %d et %d fois, attendu 1 et 1 — "+
			"un abonnement ne doit jamais en remplacer un autre", first, second)
	}
	if other != 0 {
		t.Errorf("un consommateur d'un AUTRE type a été appelé %d fois", other)
	}

	// Puis l'AUTRE type. Sans ce second envoi, le test resterait vert sur un bus
	// qui ne livrerait QU'AU premier type enregistré : « rien n'est arrivé à
	// `other` » se démontre aussi bien par un routage correct que par un routage
	// mort. Il faut prouver que la route existe avant de prouver qu'elle est
	// étanche.
	if err := bus.Publish(context.Background(), envelope("invoice.issued.v1")); err != nil {
		t.Fatalf("publication du second type en échec: %v", err)
	}
	if other != 1 {
		t.Errorf("le consommateur de invoice.issued.v1 a été appelé %d fois, attendu 1", other)
	}
	if first != 1 || second != 1 {
		t.Errorf("les consommateurs de user.registered.v1 ont reçu un événement d'un AUTRE type "+
			"(%d et %d) — la facturation s'exécuterait sur une inscription", first, second)
	}
}
