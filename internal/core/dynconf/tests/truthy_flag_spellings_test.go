package tests

import (
	"context"
	"testing"
)

// TestTruthyFlagSpellings: every driver must answer the same way to these
// spellings.
//
// This is possible because the interpretation lives in the domain
// (domain.ParseFlag) and nowhere else. A driver that redid this decoding would
// diverge at the first `"TRUE"`: changing driver would then change the
// behaviour of the application, and substitutability would be lost.
func TestTruthyFlagSpellings(t *testing.T) {
	t.Parallel()

	for _, value := range []any{true, "true", "TRUE", "True", "t", "1", 1} {
		mod := newFileModule(t, map[string]any{
			"flags": map[string]any{"flag": value},
		})
		if !mod.IsEnabled(context.Background(), "flag") {
			t.Errorf("value %#v should activate the flag", value)
		}
	}
}
