package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

// TestUnsharedDriverRefusesASeparateDispatcher: an unshared driver refuses a
// dispatcher launched in ANOTHER process.
//
// # The defect this refusal prevents
//
// The `memory` driver lives inside the process. A dispatcher launched as a
// separate binary would therefore query its own store — empty — while the
// events written by the server would stay in the server's memory.
//
// It would run publishing nothing AND without any error: clean logs, green
// probe, live process. The defect would only be discovered the day someone asks
// why a customer never received their email.
//
// Deny by default all the way down to the unknown value: a driver we do not
// know is deemed NOT shared. Erring in that direction fails a startup; erring
// in the other makes a dispatcher run empty for months.
func TestUnsharedDriverRefusesASeparateDispatcher(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"memory, not shared across processes": "memory",
		"missing driver":                      "",
		"unknown driver":                      "cassandra",
	}
	for name, driver := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := outbox.RequireSharedDriver(driver)
			if !errors.Is(err, outbox.ErrNotShared) {
				t.Fatalf("RequireSharedDriver(%q) = %v, want an explicit refusal", driver, err)
			}
			if outbox.SharedAcrossProcesses(driver) {
				t.Errorf("SharedAcrossProcesses(%q) = true, want false", driver)
			}
		})
	}

	if err := outbox.RequireSharedDriver("postgres"); err != nil {
		t.Errorf("the postgres driver is shared and must be accepted, got: %v", err)
	}
	if !outbox.SharedAcrossProcesses("postgres") {
		t.Error("SharedAcrossProcesses(\"postgres\") must be true")
	}
}
