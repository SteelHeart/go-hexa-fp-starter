package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestDisabledModuleRefusesLoudly: a disabled audit must fail when called.
//
// The inert fallback would be the worst possible choice here: a sensitive
// operation would go through believing it left a trace, and the absence of that
// trace would only be discovered the moment it is needed — that is to say, too
// late.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := audit.New(config.Module{Enabled: false, Driver: "log"}, audit.Deps{})
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	if err := mod.Record(context.Background(), completeEntry()); !errors.Is(err, audit.ErrDisabled) {
		t.Errorf("Record = %v, want ErrDisabled", err)
	}
}
