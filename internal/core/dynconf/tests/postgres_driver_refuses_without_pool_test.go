package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestPostgresDriverRefusesWithoutPool: a driver that requires a database with
// no database refuses at STARTUP, never at the first read. Discovering the
// missing dependency in production, at the first flag evaluated, would be the
// worst moment.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(config.Module{Enabled: true, Driver: "postgres"}, dynconf.Deps{})
	if !errors.Is(err, dynconf.ErrPoolRequired) {
		t.Errorf("error = %v, want ErrPoolRequired", err)
	}
}
