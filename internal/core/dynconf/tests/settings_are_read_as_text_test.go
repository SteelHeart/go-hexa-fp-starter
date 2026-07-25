package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestSettingsAreReadAsText : le pilote postgres lit une colonne `text`.
//
// Pour qu'il soit substituable au pilote `file`, toute valeur doit se présenter en
// texte quelle que soit son écriture en YAML. Sans cette normalisation, `20` lu
// depuis un fichier et `"20"` lu depuis la base donneraient deux types différents
// à l'appelant, et changer de pilote casserait le code appelant.
func TestSettingsAreReadAsText(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"settings": map[string]any{
			"entier":  20,
			"decimal": 1.5,
			"booleen": true,
			"texte":   "bonjour",
		},
	})
	ctx := context.Background()

	expected := map[domain.SettingKey]string{
		"entier":  "20",
		"decimal": "1.5",
		"booleen": "true",
		"texte":   "bonjour",
	}
	for key, want := range expected {
		if got := mod.GetSetting(ctx, key); got.Value != want {
			t.Errorf("%s = %q, attendu %q", key, got.Value, want)
		}
	}
}
