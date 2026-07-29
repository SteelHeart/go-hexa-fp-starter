package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestDisabledModuleFailsLoudly: a disabled module that « would work anyway » is
// a trap. A silently ignored event never signals itself.
func TestDisabledModuleFailsLoudly(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: false}, outbox.Deps{})
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("Enqueue on a disabled module: want ErrDisabled, got %v", err)
	}
	if err := mod.MarkDone(context.Background(), "x"); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("MarkDone on a disabled module: want ErrDisabled, got %v", err)
	}
}
