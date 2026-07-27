package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestInprocModeCallsTheLocalImplementation : le mode par défaut n'ajoute rien.
//
// # Ce que ce test verrouille
//
// `inproc` doit rendre l'implémentation locale TELLE QUELLE — pas un
// enrobage qui sérialise puis désérialise « pour l'uniformité ». Cette
// uniformité coûterait un aller-retour JSON à chaque appel entre deux modules du
// même binaire, c'est-à-dire dans le cas NORMAL du monolithe modulaire, celui
// que tout le monde exécutera.
//
// Le test vérifie aussi qu'un mode absent VAUT inproc : `hexa new` puis `go run`
// doit fonctionner sans qu'aucune ligne d'interopérabilité soit écrite.
func TestInprocModeCallsTheLocalImplementation(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"inproc", ""} {
		var localCalls int
		call := resolve(t, interop(mode, nil), noPublisher(t), localCaller(&localCalls))

		got, err := call(context.Background(), request{Ref: "r-1"})
		if err != nil {
			t.Fatalf("mode=%q: appel en échec: %v", mode, err)
		}
		if localCalls != 1 {
			t.Errorf("mode=%q: implémentation locale appelée %d fois, attendu 1", mode, localCalls)
		}
		if !got.Accepted {
			t.Errorf("mode=%q: réponse = %+v, l'implémentation locale n'a pas été jointe", mode, got)
		}
	}
}

// TestModeIsReportedForLogging : le mode retenu est lisible.
//
// Sans cela, savoir si un binaire appelle en local ou par le réseau demande de
// relire la configuration fusionnée — au moment où l'on cherche justement
// pourquoi un appel n'arrive pas.
func TestModeIsReportedForLogging(t *testing.T) {
	t.Parallel()

	bus := modulebus.New(interop("event", nil), noPublisher(t))
	if got := bus.Mode(someModule); got != modulebus.ModeEvent {
		t.Errorf("Mode(%q) = %q, attendu %q", someModule, got, modulebus.ModeEvent)
	}
}
