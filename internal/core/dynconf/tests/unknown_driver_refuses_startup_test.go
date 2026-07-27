package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
//
// La validation de configuration a déjà rejeté le pilote inconnu ; ce second refus
// garantit qu'aucun chemin — un appelant qui construit le module à la main, par
// exemple — ne contourne le premier.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(config.Module{Enabled: true, Driver: "consul"}, dynconf.Deps{})
	if err == nil {
		t.Fatal("un pilote inconnu doit refuser le démarrage")
	}
}
