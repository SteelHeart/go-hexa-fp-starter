package tests

import "testing"

// TestEachHashUsesAFreshSalt: two hashings of the SAME password give two
// different digests.
//
// Without a random salt, two accounts sharing a password would have the same
// digest: a plain `GROUP BY` query on the table would reveal who uses the same
// secret, and a rainbow table would break all those accounts in one go.
//
// It is the property most easily lost during a refactoring — a salt derived from
// the identifier "so that it is reproducible" is enough — and the most silent.
func TestEachHashUsesAFreshSalt(t *testing.T) {
	t.Parallel()

	const secret = "correct horse battery staple"
	hasher := newHasher()

	seen := map[string]struct{}{}
	for range 5 {
		encoded := hash(t, secret)
		if _, already := seen[encoded]; already {
			t.Fatal("two hashings of the same password produced the same digest")
		}
		seen[encoded] = struct{}{}

		// Every digest must stay verifiable: a random salt is useless if it is
		// not embedded in the result.
		ok, err := hasher.Verify(secret, encoded)
		if err != nil || !ok {
			t.Fatalf("digest not verifiable: ok=%v err=%v", ok, err)
		}
	}
}
