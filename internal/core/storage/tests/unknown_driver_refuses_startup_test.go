package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestUnknownDriverRefusesStartup: deny by default, right down to the factory.
//
// `s3` appears in the catalogue of intentions
// (documentation/technique/pilotes.md) but is NOT built: a heavy driver is a
// separate Go module (issue #22). The refusal must therefore be outright, never
// a silent fallback on `disk` — which would write the objects to a local disk
// that nobody backs up.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"s3", "gcs", "azure-blob", "sftp", "anything at all"} {
		if _, err := storage.New(
			config.Module{Enabled: true, Driver: driver}, storage.Deps{},
		); err == nil {
			t.Errorf("the %q driver is not built: it must refuse to start", driver)
		}
	}
}
