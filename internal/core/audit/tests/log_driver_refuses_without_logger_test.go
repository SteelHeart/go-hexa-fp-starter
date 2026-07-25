package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestLogDriverRefusesWithoutLogger : le pilote par défaut n'exige aucun SERVICE
// externe, mais il exige un journal explicite. Se rabattre sur slog.Default()
// enverrait l'audit vers une sortie que personne n'a choisi de collecter.
func TestLogDriverRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	_, err := audit.New(config.Module{Enabled: true, Driver: "log"}, audit.Deps{})
	if !errors.Is(err, audit.ErrLoggerRequired) {
		t.Errorf("erreur = %v, attendu ErrLoggerRequired", err)
	}
}
