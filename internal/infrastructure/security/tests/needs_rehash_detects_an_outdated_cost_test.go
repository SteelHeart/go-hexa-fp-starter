package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestNeedsRehashDetectsAnOutdatedCost: a digest produced with a cost lower than
// the current one must be redone.
//
// This is what makes it possible to raise the cost without invalidating existing
// accounts: after a successful verification, the password is rehashed with
// today's parameters. Without it, raising the cost would protect ONLY the new
// accounts — that is, not the ones that have existed long enough to have leaked.
//
// An unreadable digest returns `true`: better to redo it than to keep it.
func TestNeedsRehashDetectsAnOutdatedCost(t *testing.T) {
	t.Parallel()

	old := hash(t, "correct horse battery staple")

	sameCost := newHasher()
	if sameCost.NeedsRehash(old) {
		t.Error("a digest produced at the current cost must not be redone")
	}

	costlier := security.NewHasher(security.Argon2Params{
		MemoryKiB: testParams().MemoryKiB * 4, Iterations: 2, Threads: 1,
	})
	if !costlier.NeedsRehash(old) {
		t.Error("a digest produced at a lower cost must be redone")
	}

	if !sameCost.NeedsRehash("unreadable digest") {
		t.Error("an unreadable digest must be redone, not kept")
	}
}
