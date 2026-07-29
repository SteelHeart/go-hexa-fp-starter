package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestDisabledModuleRefusesLoudly: a disabled module fails explicitly. An inert
// idempotency would let duplicates through without reporting itself.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := idempotency.New(config.Module{Enabled: false, Driver: "memory"}, idempotency.Deps{})
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", "payload")); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Reserve = %v, want ErrDisabled", err)
	}
	if err := mod.Complete(ctx, "k1", nil); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Complete = %v, want ErrDisabled", err)
	}
	if err := mod.Release(ctx, "k1"); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Release = %v, want ErrDisabled", err)
	}
	if _, err := mod.Purge(ctx); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Purge = %v, want ErrDisabled", err)
	}
}
