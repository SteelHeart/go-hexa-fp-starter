package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFromPointerNeverDereferencesNil: FromPointer is the boundary conversion
// point.
//
// Past it, the domain no longer handles pointers — hence no nil access is
// possible any more. The test checks it does not dereference nil: if it did, the
// conversion meant to REMOVE that class of defects would itself be its last
// refuge.
func TestFromPointerNeverDereferencesNil(t *testing.T) {
	t.Parallel()

	var missing *string
	converted := fp.FromPointer(missing) // must not panic
	if converted.IsSome() {
		t.Error("a nil pointer must become None")
	}

	value := "present"
	held := fp.FromPointer(&value)
	if !held.IsSome() {
		t.Fatal("a non-nil pointer must become Some")
	}
	if got, _ := held.Get(); got != "present" {
		t.Errorf("value = %q", got)
	}

	// The copy is made: mutating the source must not alter the Option.
	value = "mutated"
	if got, _ := held.Get(); got != "present" {
		t.Errorf("the Option followed the source mutation: %q", got)
	}
}
