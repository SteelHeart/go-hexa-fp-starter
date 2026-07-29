package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedDriverOptionsParse: a driver option is only validated at the
// construction of the module. Without this test, `ttl: 24` — an integer instead
// of a duration, hence 24 seconds instead of 24 hours — would stay invisible
// until a legitimate replay was refused in production.
func TestShippedDriverOptionsParse(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("loading: %v", err)
	}

	ttl, err := cfg.Modules.Get("idempotency").DurationOption("ttl", 0)
	if err != nil {
		t.Fatalf("options of the idempotency module are unreadable: %v", err)
	}
	if ttl < time.Hour {
		t.Errorf("shipped ttl = %v: a replay window that short betrays a forgotten unit", ttl)
	}
}
