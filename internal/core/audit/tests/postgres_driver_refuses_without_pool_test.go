package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestPostgresDriverRefusesWithoutPool: the refusal is at startup, never at the
// first write.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := audit.New(config.Module{Enabled: true, Driver: "postgres"}, audit.Deps{})
	if !errors.Is(err, audit.ErrPoolRequired) {
		t.Errorf("error = %v, want ErrPoolRequired", err)
	}
}
