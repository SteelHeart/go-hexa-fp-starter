package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
//
// `s3` figure dans le catalogue d'intentions (documentation/technique/pilotes.md)
// mais n'est PAS construit : un pilote lourd est un module Go séparé (issue #22).
// Le refus doit donc être franc, jamais un repli silencieux sur `disk` — qui
// écrirait les objets sur un disque local que personne ne sauvegarde.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"s3", "gcs", "azure-blob", "sftp", "n'importe quoi"} {
		if _, err := storage.New(
			config.Module{Enabled: true, Driver: driver}, storage.Deps{},
		); err == nil {
			t.Errorf("le pilote %q n'est pas construit : il doit refuser le démarrage", driver)
		}
	}
}
