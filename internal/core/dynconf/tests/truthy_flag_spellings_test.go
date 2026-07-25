package tests

import (
	"context"
	"testing"
)

// TestTruthyFlagSpellings : tous les pilotes doivent répondre pareil à ces
// écritures.
//
// C'est possible parce que l'interprétation vit dans le domaine
// (domain.ParseFlag) et nulle part ailleurs. Un pilote qui referait ce décodage
// divergerait au premier `"TRUE"` : changer de pilote changerait alors le
// comportement de l'application, et la substituabilité serait perdue.
func TestTruthyFlagSpellings(t *testing.T) {
	t.Parallel()

	for _, value := range []any{true, "true", "TRUE", "True", "t", "1", 1} {
		mod := newFileModule(t, map[string]any{
			"flags": map[string]any{"drapeau": value},
		})
		if !mod.IsEnabled(context.Background(), "drapeau") {
			t.Errorf("valeur %#v devrait activer le drapeau", value)
		}
	}
}
