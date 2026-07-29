package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	modules := config.Modules{"outbox": {Enabled: true}, "audit": {Enabled: false}}
	if !modules.IsEnabled("outbox") {
		t.Error("outbox should be active")
	}
	if modules.IsEnabled("audit") {
		t.Error("audit should be inactive")
	}
	if modules.IsEnabled("nonexistent") {
		t.Error("an unknown module cannot be active")
	}
}
