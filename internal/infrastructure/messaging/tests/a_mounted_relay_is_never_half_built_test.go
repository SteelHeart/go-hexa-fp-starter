package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAMountedRelayIsNeverHalfBuilt : les trois faces d'un Broker sont toujours là.
//
// # Pourquoi ce test existe
//
// `New` rendait autrefois quatre valeurs séparées. L'appelant pouvait en oublier
// une — typiquement Close — et l'oubli ne se voit qu'au moment où les connexions
// au broker fuient en production, des semaines plus tard.
//
// Le type `Broker` a supprimé cette possibilité, mais seulement si CHAQUE relais
// remplit les trois champs. Un relais sans ressource externe est justement celui
// qu'on est tenté de laisser avec un Close nil : ce test l'interdit, parce que
// l'appelant ne doit jamais avoir à tester avant d'appeler.
func TestAMountedRelayIsNeverHalfBuilt(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{
		string(messaging.DriverInproc),
		string(messaging.DriverNoop),
	} {
		broker := mustBroker(t, driver)

		if broker.Publish == nil {
			t.Errorf("%s: Publish est nil", driver)
		}
		if broker.Consume == nil {
			t.Errorf("%s: Consume est nil", driver)
		}
		if broker.Close == nil {
			t.Fatalf("%s: Close est nil — l'appelant ne doit pas avoir à tester", driver)
		}
		if err := broker.Close(); err != nil {
			t.Errorf("%s: Close d'un relais sans ressource externe a rendu %v", driver, err)
		}
	}
}
