package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestIdentityIsNeutral: Identity returns its argument, and serves as a neutral
// branch.
//
// It looks useless until the first Fold one of whose branches has nothing to do.
// Writing it by hand every time would produce anonymous closures that all look
// alike and that one eventually mixes up.
func TestIdentityIsNeutral(t *testing.T) {
	t.Parallel()

	if got := fp.Identity(42); got != 42 {
		t.Errorf("Identity(42) = %d", got)
	}
	if got := fp.Identity("text"); got != "text" {
		t.Errorf("Identity(\"text\") = %q", got)
	}

	// Composed with itself or with another function, it stays neutral.
	composed := fp.Pipe2(fp.Identity[int], double)
	if got := composed(5); got != 10 {
		t.Errorf("Identity inside a composition = %d, want 10", got)
	}
}
