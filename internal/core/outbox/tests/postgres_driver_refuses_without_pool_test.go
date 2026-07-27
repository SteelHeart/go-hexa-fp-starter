package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	// Refus au DÉMARRAGE, pas à la première requête : un service à moitié
	// configuré échoue plus tard, ailleurs, et pour une raison sans rapport.
	_, err := outbox.New(config.Module{Enabled: true, Driver: "postgres"}, outbox.Deps{})
	if !errors.Is(err, outbox.ErrPoolRequired) {
		t.Errorf("attendu ErrPoolRequired, obtenu: %v", err)
	}
}
