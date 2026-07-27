package tests

import (
	"context"
	"testing"
)

// TestUnreadableFlagIsDenied : une valeur illisible vaut « éteint ».
//
// Une faute de frappe dans un drapeau ne doit jamais allumer la fonctionnalité.
// C'est la direction du repli qui compte : se tromper vers `false` fait manquer une
// fonctionnalité, se tromper vers `true` la livre avant l'heure.
func TestUnreadableFlagIsDenied(t *testing.T) {
	t.Parallel()

	cases := map[string]any{
		"texte quelconque":  "peut-être",
		"chaîne vide":       "",
		"nombre hors 0 / 1": 7,
		"faux explicite":    false,
		"zéro":              0,
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := newFileModule(t, map[string]any{
				"flags": map[string]any{"drapeau": value},
			})
			if mod.IsEnabled(context.Background(), "drapeau") {
				t.Errorf("valeur %#v ne doit pas activer le drapeau", value)
			}
		})
	}
}
