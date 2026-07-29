package tests

import (
	"context"
	"testing"
)

// TestUnreadableFlagIsDenied: an unreadable value counts as "switched off".
//
// A typo in a flag must never switch the feature on. It is the direction of the
// fallback that matters: erring towards `false` makes a feature go missing,
// erring towards `true` ships it before its time.
func TestUnreadableFlagIsDenied(t *testing.T) {
	t.Parallel()

	cases := map[string]any{
		"arbitrary text":       "maybe",
		"empty string":         "",
		"number outside 0 / 1": 7,
		"explicit false":       false,
		"zero":                 0,
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := newFileModule(t, map[string]any{
				"flags": map[string]any{"flag": value},
			})
			if mod.IsEnabled(context.Background(), "flag") {
				t.Errorf("value %#v must not activate the flag", value)
			}
		})
	}
}
