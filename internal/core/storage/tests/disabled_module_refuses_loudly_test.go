package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestDisabledModuleRefusesLoudly: a disabled storage fails when called.
//
// The inert fallback would be outright data loss here: a "successful" upload
// that writes nothing, and a user who discovers their document is missing weeks
// later.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := storage.New(config.Module{Enabled: false, Driver: "disk"}, storage.Deps{})
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	ctx := context.Background()

	if _, err := mod.Put(ctx, object("doc.pdf", "x")); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Put = %v, want ErrDisabled", err)
	}
	if _, err := mod.Get(ctx, "ab/cd/ef-doc.pdf"); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Get = %v, want ErrDisabled", err)
	}
	if err := mod.Delete(ctx, "ab/cd/ef-doc.pdf"); !errors.Is(err, storage.ErrDisabled) {
		t.Errorf("Delete = %v, want ErrDisabled", err)
	}
}
