package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestUnknownDriverRefusesStartup: deny by default all the way into the factory.
// Configuration validation has already rejected the driver; this second refusal
// guarantees that no path bypasses the first one.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memcached"}, idempotency.Deps{})
	if err == nil {
		t.Fatal("an unknown driver must refuse startup")
	}
}
