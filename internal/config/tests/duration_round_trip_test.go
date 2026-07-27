package tests

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationRoundTrip : la configuration effective doit pouvoir être réaffichée
// telle qu'elle serait écrite, pour qu'un diff de configuration soit lisible.
func TestDurationRoundTrip(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 1h30m0s`), &out); err != nil {
		t.Fatalf("décodage: %v", err)
	}
	encoded, err := yaml.Marshal(out)
	if err != nil {
		t.Fatalf("encodage: %v", err)
	}

	var back struct{ D config.Duration }
	if err := yaml.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("re-décodage de %q: %v", encoded, err)
	}
	if back.D != out.D {
		t.Errorf("aller-retour non conservatif: %v puis %v", out.D, back.D)
	}
}
