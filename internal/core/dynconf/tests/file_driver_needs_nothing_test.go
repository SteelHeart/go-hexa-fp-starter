package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestFileDriverNeedsNothing verrouille la promesse de l'ADR 012 : le pilote par
// défaut ne réclame ni base, ni journal, ni horloge.
func TestFileDriverNeedsNothing(t *testing.T) {
	t.Parallel()

	mod, err := dynconf.New(config.Module{Enabled: true, Driver: "file"}, dynconf.Deps{})
	if err != nil {
		t.Fatalf("le pilote par défaut ne doit réclamer aucune dépendance: %v", err)
	}
	if mod.IsEnabled(context.Background(), "quelconque") {
		t.Error("sans valeur déclarée, tout drapeau doit être inactif")
	}
}
