package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestIntOption : zéro et les valeurs négatives sont refusés.
//
// Aucune option entière du socle n'a de sens à zéro : un lot de zéro message ou
// zéro tentative autorisée décrivent un composant qui tourne sans rien faire. Le
// refus au démarrage vaut mieux qu'un dépileur qui scrute indéfiniment un magasin
// dont il ne sortira jamais rien.
func TestIntOption(t *testing.T) {
	t.Parallel()

	const fallback = 50

	cases := map[string]struct {
		options map[string]any
		want    int
		refused bool
	}{
		"absente : valeur par défaut": {options: nil, want: fallback},
		"nulle : valeur par défaut":   {options: map[string]any{"batch_size": nil}, want: fallback},
		"entier":                      {options: map[string]any{"batch_size": 20}, want: 20},
		"entier 64 bits":              {options: map[string]any{"batch_size": int64(30)}, want: 30},
		"zéro : refus":                {options: map[string]any{"batch_size": 0}, refused: true},
		"négatif : refus":             {options: map[string]any{"batch_size": -1}, refused: true},
		"chaîne : refus":              {options: map[string]any{"batch_size": "50"}, refused: true},
		"décimal : refus":             {options: map[string]any{"batch_size": 1.5}, refused: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.IntOption("batch_size", fallback)
			if tc.refused {
				if err == nil {
					t.Fatalf("attendu un refus, reçu %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("refus inattendu: %v", err)
			}
			if got != tc.want {
				t.Errorf("IntOption = %d, attendu %d", got, tc.want)
			}
		})
	}
}
