package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestDisabledModuleRefusesLoudly : un stockage désactivé échoue à l'appel.
//
// Le repli inerte serait ici une perte de données franche : un téléversement
// « réussi » qui n'écrit rien, et un utilisateur qui découvre l'absence de son
// document des semaines plus tard.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := storage.New(config.Module{Enabled: false, Driver: "disk"}, storage.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	ctx := context.Background()

	if _, err := mod.Put(ctx, object("doc.pdf", "x")); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Put = %v, attendu ErrDisabled", err)
	}
	if _, err := mod.Get(ctx, "ab/cd/ef-doc.pdf"); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Get = %v, attendu ErrDisabled", err)
	}
	if err := mod.Delete(ctx, "ab/cd/ef-doc.pdf"); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Delete = %v, attendu ErrDisabled", err)
	}
}
