package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestDiskDriverNeedsNothing locks down the promise of ADR 012: the default
// driver claims neither a database, nor a bucket, nor a provider client.
//
// The base directory is created at startup rather than at the first upload:
// discovering an unusable store when a user sends their first file would be the
// worst moment.
func TestDiskDriverNeedsNothing(t *testing.T) {
	t.Parallel()

	mod, err := storage.New(config.Module{
		Enabled: true,
		Driver:  "disk",
		Options: map[string]any{"base_dir": t.TempDir()},
	}, storage.Deps{})
	if err != nil {
		t.Fatalf("the default driver must claim no dependency: %v", err)
	}

	if _, err := mod.Put(context.Background(), object("first.txt", "content")); err != nil {
		t.Errorf("Put on the disk driver: %v", err)
	}
}
