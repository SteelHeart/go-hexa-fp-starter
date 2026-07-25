package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestDisabledModuleRefusesLoudly : un module désactivé échoue explicitement.
// Une idempotence inerte laisserait passer les doublons sans se signaler.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := idempotency.New(config.Module{Enabled: false, Driver: "memory"}, idempotency.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", "charge")); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Reserve = %v, attendu ErrDisabled", err)
	}
	if err := mod.Complete(ctx, "k1", nil); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Complete = %v, attendu ErrDisabled", err)
	}
	if err := mod.Release(ctx, "k1"); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Release = %v, attendu ErrDisabled", err)
	}
	if _, err := mod.Purge(ctx); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Purge = %v, attendu ErrDisabled", err)
	}
}
