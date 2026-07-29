package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestFileDriverNeedsNothing locks down the promise of ADR 012: the default
// driver claims neither a database, nor a logger, nor a clock.
func TestFileDriverNeedsNothing(t *testing.T) {
	t.Parallel()

	mod, err := dynconf.New(config.Module{Enabled: true, Driver: "file"}, dynconf.Deps{})
	if err != nil {
		t.Fatalf("the default driver must claim no dependency: %v", err)
	}
	if mod.IsEnabled(context.Background(), "anything") {
		t.Error("with no declared value, every flag must be inactive")
	}
}
