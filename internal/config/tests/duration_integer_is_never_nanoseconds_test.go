package tests

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationIntegerIsNeverNanoseconds: `read_timeout: 5` must mean five
// SECONDS. Read as nanoseconds, the timeout would be practically zero — hence a
// silent outage, the worst of defects.
func TestDurationIntegerIsNeverNanoseconds(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 5`), &out); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if out.D.Duration() < time.Second {
		t.Errorf("integer interpreted as nanoseconds: %v", out.D.Duration())
	}
}
