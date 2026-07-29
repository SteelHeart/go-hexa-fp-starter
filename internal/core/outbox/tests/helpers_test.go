// Package tests contains the BLACK BOX tests of the outbox module: they only
// use the public API, exactly like a caller.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is checked, without having to open it.
package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

// fixedNow makes the tests deterministic: no real clock, no waiting.
func fixedNow() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

func newMemoryModule(t *testing.T) outbox.Module {
	t.Helper()
	// Deps without Pool: this is exactly the promise of ADR 012.
	mod, err := outbox.New(
		config.Module{Enabled: true, Driver: "memory"},
		outbox.Deps{Now: fixedNow},
	)
	if err != nil {
		t.Fatalf("building the module on the memory driver: %v", err)
	}
	return mod
}
