package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique. La
// validation de configuration a déjà rejeté le pilote ; ce second refus garantit
// qu'aucun chemin ne contourne le premier.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memcached"}, idempotency.Deps{})
	if err == nil {
		t.Fatal("un pilote inconnu doit refuser le démarrage")
	}
}
