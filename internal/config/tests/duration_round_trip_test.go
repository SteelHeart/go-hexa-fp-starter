package tests

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationRoundTrip: the effective configuration must be redisplayable
// exactly as it would be written, so that a configuration diff is readable.
func TestDurationRoundTrip(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 1h30m0s`), &out); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	encoded, err := yaml.Marshal(out)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}

	var back struct{ D config.Duration }
	if err := yaml.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("re-decoding of %q: %v", encoded, err)
	}
	if back.D != out.D {
		t.Errorf("round trip is not conservative: %v then %v", out.D, back.D)
	}
}
