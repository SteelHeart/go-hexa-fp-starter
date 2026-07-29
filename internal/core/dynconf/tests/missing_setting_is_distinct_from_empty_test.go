package tests

import (
	"context"
	"testing"
)

// TestMissingSettingIsDistinctFromEmpty: "absent" and "present and empty" are
// two different situations.
//
// Confusing them would apply a default value in place of a deliberate
// operations decision — emptying a banner to make it disappear would make it
// reappear with its original text.
func TestMissingSettingIsDistinctFromEmpty(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"settings": map[string]any{"banner": ""},
	})
	ctx := context.Background()

	empty := mod.GetSetting(ctx, "banner")
	if !empty.Found {
		t.Error("a setting declared empty must be found")
	}
	if empty.Value != "" {
		t.Errorf("value = %q, want empty", empty.Value)
	}

	if absent := mod.GetSetting(ctx, "never_set"); absent.Found {
		t.Error("a setting never set must not be found")
	}
}
