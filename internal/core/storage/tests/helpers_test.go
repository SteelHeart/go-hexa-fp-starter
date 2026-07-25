// Package tests contient les tests en BOÎTE NOIRE du module storage : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// baseURL est l'adresse de lecture utilisée par les tests.
const baseURL = "/files"

// newDiskModule construit le module sur son pilote par défaut, dans un répertoire
// temporaire nettoyé automatiquement.
func newDiskModule(t *testing.T) storage.Module {
	t.Helper()
	mod, err := storage.New(config.Module{
		Enabled: true,
		Driver:  "disk",
		Options: map[string]any{"base_dir": t.TempDir(), "base_url": baseURL},
	}, storage.Deps{})
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod
}

// object forge un objet à stocker.
func object(name, content string) domain.Object {
	return domain.Object{
		Name:        name,
		ContentType: "text/plain",
		Content:     strings.NewReader(content),
	}
}
