package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestPostgresDriverRefusesWithoutPool : un pilote qui exige une base sans base
// refuse au DÉMARRAGE, jamais à la première lecture. Découvrir la dépendance
// manquante en production, au premier drapeau évalué, serait le pire moment.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(config.Module{Enabled: true, Driver: "postgres"}, dynconf.Deps{})
	if !errors.Is(err, dynconf.ErrPoolRequired) {
		t.Errorf("erreur = %v, attendu ErrPoolRequired", err)
	}
}
