package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestPostgresDriverRefusesWithoutPool : un pilote qui exige une base sans base
// refuse au démarrage, jamais à la première requête.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "postgres"}, idempotency.Deps{})
	if !errors.Is(err, idempotency.ErrPoolRequired) {
		t.Errorf("erreur = %v, attendu ErrPoolRequired", err)
	}
}
