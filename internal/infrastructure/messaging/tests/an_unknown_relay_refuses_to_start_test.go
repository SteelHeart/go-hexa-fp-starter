package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnUnknownRelayRefusesToStart : deny par défaut sur le choix du transport.
//
// # Le défaut que ce test attrape
//
// Un relais inconnu qui se rabattrait sur « noop » produirait le pire défaut
// possible de ce socle : le service démarre, répond, l'outbox se remplit, le
// dépileur la vide, tout est marqué publié — et AUCUN événement ne sort. Le
// courriel de bienvenue ne part jamais, et rien, nulle part, ne le signale.
//
// Une faute de frappe dans `messaging.driver` suffit. Elle doit coûter un refus
// de démarrage bruyant, jamais une perte silencieuse.
func TestAnUnknownRelayRefusesToStart(t *testing.T) {
	t.Parallel()

	// Y compris des valeurs qui « ressemblent » : un repli sur le plus proche
	// serait une interprétation, et on n'interprète pas une configuration.
	for _, driver := range []string{"", "kafka2", "Inproc", "in-proc", "amqp", "none"} {
		if _, err := messaging.New(relayConfig(driver), quietLogger()); err == nil {
			t.Errorf("driver=%q accepté — un relais non prévu doit refuser le démarrage", driver)
		} else if !errors.Is(err, messaging.ErrUnknownDriver) {
			t.Errorf("driver=%q rendu avec %v, attendu ErrUnknownDriver", driver, err)
		}
	}
}
