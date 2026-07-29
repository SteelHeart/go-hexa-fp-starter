package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestSettingsAreReadAsText: the postgres driver reads a `text` column.
//
// For it to be substitutable for the `file` driver, every value must present
// itself as text whatever its spelling in YAML. Without that normalisation,
// `20` read from a file and `"20"` read from the database would give two
// different types to the caller, and changing driver would break the calling
// code.
func TestSettingsAreReadAsText(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"settings": map[string]any{
			"integer": 20,
			"decimal": 1.5,
			"boolean": true,
			"text":    "hello",
		},
	})
	ctx := context.Background()

	expected := map[domain.SettingKey]string{
		"integer": "20",
		"decimal": "1.5",
		"boolean": "true",
		"text":    "hello",
	}
	for key, want := range expected {
		if got := mod.GetSetting(ctx, key); got.Value != want {
			t.Errorf("%s = %q, want %q", key, got.Value, want)
		}
	}
}
