package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	modules := config.Modules{"outbox": {Enabled: true}, "audit": {Enabled: false}}
	if !modules.IsEnabled("outbox") {
		t.Error("outbox devrait être actif")
	}
	if modules.IsEnabled("audit") {
		t.Error("audit devrait être inactif")
	}
	if modules.IsEnabled("inexistant") {
		t.Error("un module inconnu ne peut pas être actif")
	}
}
