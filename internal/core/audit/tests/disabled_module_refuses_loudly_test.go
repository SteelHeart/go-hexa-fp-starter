package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestDisabledModuleRefusesLoudly : un audit désactivé doit échouer à l'appel.
//
// Le repli inerte serait ici le pire choix possible : une opération sensible
// passerait en croyant laisser une trace, et l'absence de trace ne se découvrirait
// qu'au moment où on en a besoin — c'est-à-dire trop tard.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := audit.New(config.Module{Enabled: false, Driver: "log"}, audit.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	if err := mod.Record(context.Background(), completeEntry()); !errors.Is(err, audit.ErrDisabled) {
		t.Errorf("Record = %v, attendu ErrDisabled", err)
	}
}
