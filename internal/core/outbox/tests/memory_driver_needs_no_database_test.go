package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestMemoryDriverNeedsNoDatabase locks down the central promise of the
// starter: the default driver starts without any connection.
func TestMemoryDriverNeedsNoDatabase(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: true, Driver: "memory"}, outbox.Deps{})
	if err != nil {
		t.Fatalf("the memory driver must require no dependency: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); err != nil {
		t.Errorf("Enqueue without a database must succeed: %v", err)
	}
}
