package tests

import (
	"testing"
)

// TestStepsRunInTheDeclaredOrder: the pipeline runs in the order in which it
// reads.
//
// That is the promise of the style: the body of `NewRegisterUser` reads like the
// list of business steps, and that reading must tell the truth. An order
// different from the one in the code would make the use case unreadable — one
// would only think one understands.
//
// Two positions have, in addition, a SECURITY or a COST reason:
//   - the availability of the address is checked before hashing (cost);
//   - the event is recorded after the write (no ghost event).
func TestStepsRunInTheDeclaredOrder(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	_ = userOf(t, register(nominalDeps(observed)))

	want := []string{
		"EmailIsTaken",
		"HashPassword",
		"GenerateID",
		"Now",
		"SaveUser",
		"PublishEvent",
	}

	if len(observed.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", observed.calls, want)
	}
	for i, name := range want {
		if observed.calls[i] != name {
			t.Fatalf("calls = %v, want %v", observed.calls, want)
		}
	}
}
