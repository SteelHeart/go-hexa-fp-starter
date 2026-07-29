package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestPostgresDriverRefusesWithoutPool: a driver that requires a database
// without a database refuses at startup, never on the first request.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "postgres"}, idempotency.Deps{})
	if !errors.Is(err, idempotency.ErrPoolRequired) {
		t.Errorf("error = %v, want ErrPoolRequired", err)
	}
}
