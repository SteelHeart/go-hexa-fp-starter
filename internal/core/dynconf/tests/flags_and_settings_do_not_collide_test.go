package tests

import (
	"context"
	"testing"
)

// TestFlagsAndSettingsDoNotCollide: the two natures share key names without
// getting mixed up.
//
// Without qualification by nature (domain.Qualify), a setting named like a flag
// would be read as a boolean: `mode: "strict"` would become a switched-off
// flag, since "strict" is not a readable boolean. The feature would be
// disabled by a setting that has nothing to do with it.
func TestFlagsAndSettingsDoNotCollide(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"flags":    map[string]any{"mode": true},
		"settings": map[string]any{"mode": "strict"},
	})
	ctx := context.Background()

	if !mod.IsEnabled(ctx, "mode") {
		t.Error("the `mode` flag must be active")
	}
	if got := mod.GetSetting(ctx, "mode"); got.Value != "strict" {
		t.Errorf("setting `mode` = %q, want \"strict\"", got.Value)
	}
}
