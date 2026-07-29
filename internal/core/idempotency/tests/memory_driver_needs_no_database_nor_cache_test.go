package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestMemoryDriverNeedsNoDatabaseNorCache locks down the promise of ADR 012:
// `hexa new` then `go run` must start without a database, without Redis,
// without Docker.
func TestMemoryDriverNeedsNoDatabaseNorCache(t *testing.T) {
	t.Parallel()

	mod, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memory"},
		idempotency.Deps{}, // no Pool, no Cache, no clock
	)
	if err != nil {
		t.Fatalf("the default driver must claim no dependency: %v", err)
	}
	if _, err := mod.Reserve(context.Background(), request("k1", "payload")); err != nil {
		t.Errorf("reservation on the memory driver: %v", err)
	}
}
