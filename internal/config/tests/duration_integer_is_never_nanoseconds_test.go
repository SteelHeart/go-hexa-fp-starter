package tests

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationIntegerIsNeverNanoseconds : `read_timeout: 5` doit valoir cinq
// SECONDES. Lu en nanosecondes, le délai vaudrait pratiquement zéro — donc une
// panne silencieuse, le pire des défauts.
func TestDurationIntegerIsNeverNanoseconds(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 5`), &out); err != nil {
		t.Fatalf("décodage: %v", err)
	}
	if out.D.Duration() < time.Second {
		t.Errorf("entier interprété en nanosecondes: %v", out.D.Duration())
	}
}
