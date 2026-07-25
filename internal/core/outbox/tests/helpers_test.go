// Package tests contient les tests en BOÎTE NOIRE du module outbox : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

// fixedNow rend les tests déterministes : ni horloge réelle, ni attente.
func fixedNow() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

func newMemoryModule(t *testing.T) outbox.Module {
	t.Helper()
	// Deps sans Pool : c'est exactement la promesse de l'ADR 012.
	mod, err := outbox.New(
		config.Module{Enabled: true, Driver: "memory"},
		outbox.Deps{Now: fixedNow},
	)
	if err != nil {
		t.Fatalf("construction du module en pilote memory: %v", err)
	}
	return mod
}
