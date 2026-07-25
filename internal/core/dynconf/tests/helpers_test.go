// Package tests contient les tests en BOÎTE NOIRE du module dynconf : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// newFileModule construit le module sur son pilote par défaut, avec les valeurs
// données en options — exactement comme le fera config/modules.yaml.
func newFileModule(t *testing.T, options map[string]any) dynconf.Module {
	t.Helper()
	mod, err := dynconf.New(
		config.Module{Enabled: true, Driver: "file", Options: options},
		dynconf.Deps{},
	)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod
}
