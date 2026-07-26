package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestAnUnknownTransportRefusesToResolve : deny par défaut sur le transport.
//
// # Le défaut que ce test attrape
//
// Un mode inconnu qui se rabattrait sur `inproc` — le mode le moins coûteux,
// donc le repli « raisonnable » — produirait un appel LOCAL là où l'exploitant
// a demandé un appel distant. Les deux modules sont déployés séparément, le
// module appelant exécute silencieusement sa propre copie de la capacité, et
// tout fonctionne : les données partent simplement dans la mauvaise base.
//
// C'est le pire cas d'un repli silencieux — il n'échoue pas, il se trompe.
func TestAnUnknownTransportRefusesToResolve(t *testing.T) {
	t.Parallel()

	// Y compris les valeurs qui ressemblent à un mode valide : on ne devine pas
	// une intention à partir d'une faute de frappe.
	for _, mode := range []string{"grpc", "HTTP", "in-proc", "events", "off"} {
		var localCalls int

		_, err := modulebus.Resolve(
			modulebus.New(interop(mode, nil), noPublisher(t)),
			someModule, route(), someEvent, localCaller(&localCalls))

		if err == nil {
			t.Errorf("mode=%q accepté — un transport non prévu doit refuser la résolution", mode)
			continue
		}
		if !errors.Is(err, modulebus.ErrUnknownMode) {
			t.Errorf("mode=%q rendu avec %v, attendu ErrUnknownMode", mode, err)
		}
	}
}
