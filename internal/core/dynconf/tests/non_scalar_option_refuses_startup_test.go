package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestNonScalarOptionRefusesStartup : `flags: {a: {b: 1}}` est une faute de frappe,
// pas une intention. La laisser passer donnerait un drapeau silencieusement
// inactif — le défaut le plus difficile à voir sur ce module, puisqu'un drapeau
// inactif ressemble en tout point à un drapeau qu'on n'a pas encore allumé.
func TestNonScalarOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]any{
		"table imbriquée":  {"flags": map[string]any{"a": map[string]any{"b": 1}}},
		"liste":            {"settings": map[string]any{"a": []any{1, 2}}},
		"groupe non table": {"flags": "nouveau_paiement"},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := dynconf.New(
				config.Module{Enabled: true, Driver: "file", Options: options},
				dynconf.Deps{},
			)
			if err == nil {
				t.Error("une option mal formée doit refuser le démarrage")
			}
		})
	}
}
