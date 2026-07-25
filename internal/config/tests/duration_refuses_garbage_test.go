package tests

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationRefusesGarbage : une durée illisible doit ÉCHOUER, jamais valoir
// zéro. Un délai à zéro n'est pas une valeur sûre.
func TestDurationRefusesGarbage(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`d: cinq secondes`, `d: 5 secondes`, `d: [1,2]`, `d: {a: 1}`} {
		var out struct{ D config.Duration }
		if err := yaml.Unmarshal([]byte(raw), &out); err == nil {
			t.Errorf("%q accepté alors qu'il devrait échouer (obtenu %v)", raw, out.D)
		}
	}
}
