// Package tests holds the BLACK BOX tests of the dynconf module: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// newFileModule builds the module on its default driver, with the values given
// as options — exactly as config/modules.yaml will do.
func newFileModule(t *testing.T, options map[string]any) dynconf.Module {
	t.Helper()
	mod, err := dynconf.New(
		config.Module{Enabled: true, Driver: "file", Options: options},
		dynconf.Deps{},
	)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod
}
