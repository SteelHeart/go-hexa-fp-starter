package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestInprocModeCallsTheLocalImplementation: the default mode adds nothing.
//
// # What this test locks down
//
// `inproc` must return the local implementation AS IS — not a wrapper that
// serialises then deserialises "for uniformity". That uniformity would cost a
// JSON round trip on every call between two modules of the same binary, that is
// to say in the NORMAL case of the modular monolith, the one everyone will run.
//
// The test also checks that an absent mode MEANS inproc: `hexa new` then
// `go run` must work without a single line of interoperability being written.
func TestInprocModeCallsTheLocalImplementation(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"inproc", ""} {
		var localCalls int
		call := resolve(t, interop(mode, nil), noPublisher(t), localCaller(&localCalls))

		got, err := call(context.Background(), request{Ref: "r-1"})
		if err != nil {
			t.Fatalf("mode=%q: call failed: %v", mode, err)
		}
		if localCalls != 1 {
			t.Errorf("mode=%q: local implementation called %d times, want 1", mode, localCalls)
		}
		if !got.Accepted {
			t.Errorf("mode=%q: reply = %+v, the local implementation was not reached", mode, got)
		}
	}
}

// TestModeIsReportedForLogging: the selected mode is readable.
//
// Without it, knowing whether a binary calls locally or over the network
// requires re-reading the merged configuration — at the very moment one is
// looking for why a call is not arriving.
func TestModeIsReportedForLogging(t *testing.T) {
	t.Parallel()

	bus := modulebus.New(interop("event", nil), noPublisher(t))
	if got := bus.Mode(someModule); got != modulebus.ModeEvent {
		t.Errorf("Mode(%q) = %q, want %q", someModule, got, modulebus.ModeEvent)
	}
}
