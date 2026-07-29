package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestDisabledModuleDeniesReadsAndRefusesWrites documents the only exception in
// the repository to "a disabled module fails loudly".
//
// Reading can NOT fail: ports.IsEnabled does not return an error. The only
// possible answer, `false`, is precisely the "deny by default" one — inertia
// coincides with refusal here, which is the case for no other module.
//
// Writing, for its part, CAN speak: it refuses. A caller that believed it had
// changed a flag on a switched-off module is the only real trap possible here.
func TestDisabledModuleDeniesReadsAndRefusesWrites(t *testing.T) {
	t.Parallel()

	mod, err := dynconf.New(config.Module{Enabled: false, Driver: "file"}, dynconf.Deps{})
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	ctx := context.Background()

	if mod.IsEnabled(ctx, "new_payment") {
		t.Error("a disabled module must activate no flag")
	}
	if got := mod.GetSetting(ctx, "threshold"); got.Found {
		t.Error("a disabled module must find no setting")
	}

	change := domain.Change{Kind: domain.KindFlag, Key: "new_payment", Value: "true"}
	if err := mod.Set(ctx, change); !errors.Is(err, dynconf.ErrDisabled) {
		t.Errorf("Set = %v, want ErrDisabled", err)
	}

	mod.Invalidate() // must not panic on a switched-off module
}
