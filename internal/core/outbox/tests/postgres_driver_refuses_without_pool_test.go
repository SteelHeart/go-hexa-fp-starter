package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	// Refusal at STARTUP, not at the first query: a half-configured service
	// fails later, elsewhere, and for an unrelated reason.
	_, err := outbox.New(config.Module{Enabled: true, Driver: "postgres"}, outbox.Deps{})
	if !errors.Is(err, outbox.ErrPoolRequired) {
		t.Errorf("want ErrPoolRequired, got: %v", err)
	}
}
