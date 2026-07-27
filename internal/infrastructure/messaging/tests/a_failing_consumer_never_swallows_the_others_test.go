package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAFailingConsumerNeverSwallowsTheOthers : un échec ne court-circuite rien,
// et il REMONTE.
//
// # Les deux défauts que ce test attrape
//
//  1. Un `return err` dans la boucle sur les consommateurs. Le premier échec
//     empêcherait les suivants de s'exécuter — et comme l'ordre d'itération est
//     celui de l'abonnement, quel module est privé dépendrait de l'ordre de
//     montage. Un défaut qui change de victime à chaque redémarrage.
//  2. Un échec avalé. Le dépileur marquerait le message publié alors qu'un
//     consommateur ne l'a pas traité : perte définitive, sans trace, et l'outbox
//     aurait fait tout son travail pour rien.
//
// L'erreur remontée doit AUSSI rester identifiable par `errors.Is` : le dépileur
// distingue les échecs pour décider du recul, il ne lit pas des chaînes.
func TestAFailingConsumerNeverSwallowsTheOthers(t *testing.T) {
	t.Parallel()

	boom := errors.New("consommateur indisponible")
	bus := messaging.NewInproc(quietLogger())

	var afterFailure int
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		return boom
	})
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		afterFailure++
		return nil
	})

	err := bus.Publish(context.Background(), envelope("user.registered.v1"))

	if err == nil {
		t.Fatal("un consommateur en échec n'a pas fait échouer la publication — " +
			"le dépileur marquerait le message traité, et il serait perdu")
	}
	if !errors.Is(err, boom) {
		t.Errorf("l'erreur remontée = %v, elle doit rester identifiable par errors.Is", err)
	}
	if afterFailure != 1 {
		t.Errorf("le consommateur suivant a été appelé %d fois, attendu 1 — "+
			"un échec ne doit pas priver les autres de l'événement", afterFailure)
	}
}
