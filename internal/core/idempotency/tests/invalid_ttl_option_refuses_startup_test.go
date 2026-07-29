package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestInvalidTTLOptionRefusesStartup: an unreadable option refuses startup.
// Silently falling back on the default value would give a surprise TTL, hence a
// replay window that is not the one one believes in.
func TestInvalidTTLOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	cases := map[string]any{
		"missing unit": "24",
		"negative":     "-1h",
		"zero":         "0s",
		"boolean":      true,
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Module{
				Enabled: true, Driver: "memory",
				Options: map[string]any{"ttl": value},
			}
			if _, err := idempotency.New(cfg, idempotency.Deps{}); err == nil {
				t.Errorf("ttl=%v must refuse startup", value)
			}
		})
	}
}
