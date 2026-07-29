package tests

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationRefusesGarbage: an unreadable duration must FAIL, never be worth
// zero. A timeout of zero is not a safe value.
func TestDurationRefusesGarbage(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`d: five seconds`, `d: 5 seconds`, `d: [1,2]`, `d: {a: 1}`} {
		var out struct{ D config.Duration }
		if err := yaml.Unmarshal([]byte(raw), &out); err == nil {
			t.Errorf("%q accepted when it should fail (got %v)", raw, out.D)
		}
	}
}
