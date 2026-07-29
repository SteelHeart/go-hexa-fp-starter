package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestADisabledModuleFailsAtCallNotAtMount: disabled resolves, but does not call.
//
// # Why this asymmetry is intended
//
// `disabled` is the only mode whose resolution SUCCEEDS even though no call
// will ever be able to go through. This is not an inconsistency, it is the
// point: the binary must start with a capability cut off — during an incident
// at the provider's, during a migration — without having to remove code.
//
// But the cut-off must be LOUD in use. The trap would be to return the zero
// value and `nil`: the caller would read an empty record as a valid reply, and
// would treat "capability cut off" as "nothing to do". A disabled eligibility
// check would then return "not eligible" without anyone noticing.
//
// Deny by default all the way here: cut off means refused, never empty.
func TestADisabledModuleFailsAtCallNotAtMount(t *testing.T) {
	t.Parallel()

	var localCalls int
	call := resolve(t, interop("disabled", nil), noPublisher(t), localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-1"})

	if err == nil {
		t.Fatal("a disabled module returned nil — the caller would read the zero value " +
			"as a valid reply")
	}
	if !errors.Is(err, modulebus.ErrDisabled) {
		t.Errorf("error returned = %v, want ErrDisabled", err)
	}
	if got.Accepted {
		t.Errorf("non-zero reply returned along with the error: %+v", got)
	}
	if localCalls != 0 {
		t.Errorf("the local implementation was called %d times — "+
			"\"disabled\" must not fall back on the local call", localCalls)
	}
}
