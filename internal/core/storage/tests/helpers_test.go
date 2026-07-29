// Package tests holds the BLACK BOX tests of the storage module: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// baseURL is the read address used by the tests.
const baseURL = "/files"

// newDiskModule builds the module on its default driver, in a temporary
// directory cleaned up automatically.
func newDiskModule(t *testing.T) storage.Module {
	t.Helper()
	mod, err := storage.New(config.Module{
		Enabled: true,
		Driver:  "disk",
		Options: map[string]any{"base_dir": t.TempDir(), "base_url": baseURL},
	}, storage.Deps{})
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod
}

// object forges an object to be stored.
func object(name, content string) domain.Object {
	return domain.Object{
		Name:        name,
		ContentType: "text/plain",
		Content:     strings.NewReader(content),
	}
}
