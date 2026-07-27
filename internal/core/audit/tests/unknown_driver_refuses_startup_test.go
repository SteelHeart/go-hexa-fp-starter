package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := audit.New(config.Module{Enabled: true, Driver: "syslog"}, audit.Deps{})
	if err == nil {
		t.Fatal("un pilote inconnu doit refuser le démarrage")
	}
}
