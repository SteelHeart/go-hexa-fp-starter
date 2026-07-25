package tests

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationAcceptsHumanForms verrouille le défaut qui rendait toute la
// configuration inutilisable : yaml.v3 ne décode pas "5s" en time.Duration, il
// n'accepte qu'un entier de nanosecondes.
func TestDurationAcceptsHumanForms(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		yaml string
		want time.Duration
	}{
		"secondes":              {yaml: `d: 5s`, want: 5 * time.Second},
		"heures et minutes":     {yaml: `d: 1h30m`, want: 90 * time.Minute},
		"millisecondes":         {yaml: `d: 250ms`, want: 250 * time.Millisecond},
		"zero explicite":        {yaml: `d: 0s`, want: 0},
		"entier lu en SECONDES": {yaml: `d: 30`, want: 30 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out struct{ D config.Duration }
			if err := yaml.Unmarshal([]byte(tc.yaml), &out); err != nil {
				t.Fatalf("décodage de %q: %v", tc.yaml, err)
			}
			if out.D.Duration() != tc.want {
				t.Errorf("%q → %v, attendu %v", tc.yaml, out.D.Duration(), tc.want)
			}
		})
	}
}
