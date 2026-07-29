package tests

import (
	"context"
	"testing"
)

// TestUnknownFlagIsDenied: this is THE guarantee of the module. A flag that
// switched itself on without having been set would put into production the
// incomplete feature it is precisely meant to hide (ADR 007, trunk-based
// development).
func TestUnknownFlagIsDenied(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"flags": map[string]any{"known": true},
	})
	ctx := context.Background()

	if !mod.IsEnabled(ctx, "known") {
		t.Error("a flag declared active must be active")
	}
	if mod.IsEnabled(ctx, "unknown") {
		t.Error("a flag never declared must be inactive")
	}
}
