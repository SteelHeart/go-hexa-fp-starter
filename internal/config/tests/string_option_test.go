package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestStringOption : une valeur présente mais vide trahit une variable
// d'environnement non substituée, pas une intention. Se rabattre sur la valeur
// par défaut masquerait une configuration cassée.
func TestStringOption(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		options map[string]any
		want    string
		refused bool
	}{
		"absente : valeur par défaut": {options: nil, want: "idempotency"},
		"présente":                    {options: map[string]any{"namespace": "paiements"}, want: "paiements"},
		"vide : refus":                {options: map[string]any{"namespace": ""}, refused: true},
		"type inattendu : refus":      {options: map[string]any{"namespace": 42}, refused: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.StringOption("namespace", "idempotency")
			if tc.refused {
				if err == nil {
					t.Fatalf("attendu un refus, reçu %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("refus inattendu: %v", err)
			}
			if got != tc.want {
				t.Errorf("StringOption = %q, attendu %q", got, tc.want)
			}
		})
	}
}
