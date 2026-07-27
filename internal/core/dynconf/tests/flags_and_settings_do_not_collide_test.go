package tests

import (
	"context"
	"testing"
)

// TestFlagsAndSettingsDoNotCollide : les deux natures partagent des noms de clé
// sans se mélanger.
//
// Sans qualification par nature (domain.Qualify), un réglage nommé comme un
// drapeau serait lu en booléen : `mode: "strict"` deviendrait un drapeau éteint,
// puisque « strict » n'est pas un booléen lisible. La fonctionnalité serait
// désactivée par un réglage qui n'a rien à voir.
func TestFlagsAndSettingsDoNotCollide(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"flags":    map[string]any{"mode": true},
		"settings": map[string]any{"mode": "strict"},
	})
	ctx := context.Background()

	if !mod.IsEnabled(ctx, "mode") {
		t.Error("le drapeau `mode` doit être actif")
	}
	if got := mod.GetSetting(ctx, "mode"); got.Value != "strict" {
		t.Errorf("réglage `mode` = %q, attendu \"strict\"", got.Value)
	}
}
