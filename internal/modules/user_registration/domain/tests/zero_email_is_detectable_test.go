package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestZeroEmailIsDetectable: an address that was never constructed is
// recognisable.
//
// The type forbids FABRICATING an invalid address, but not declaring an empty
// one — `var e Email` remains legal in Go. `IsZero` is what allows an adapter to
// refuse a value that never went through `NewEmail`, rather than writing an
// empty string into the database.
func TestZeroEmailIsDetectable(t *testing.T) {
	t.Parallel()

	var neverConstructed domain.Email
	if !neverConstructed.IsZero() {
		t.Error("an address that was never constructed must be detectable")
	}
	if neverConstructed.String() != "" {
		t.Errorf("address never constructed = %q, want empty", neverConstructed.String())
	}

	if validEmail(t, "alice@example.com").IsZero() {
		t.Error("a constructed address must not be seen as empty")
	}
}
