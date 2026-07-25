package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	// Deny par défaut : jamais de repli sur « le pilote le plus proche ».
	if _, err := outbox.New(config.Module{Enabled: true, Driver: "postgresql"}, outbox.Deps{}); err == nil {
		t.Error("un pilote inconnu doit refuser le démarrage")
	}
}
