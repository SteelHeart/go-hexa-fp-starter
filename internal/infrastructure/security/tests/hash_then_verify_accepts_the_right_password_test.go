package tests

import "testing"

// TestHashThenVerifyAcceptsTheRightPassword: the nominal path.
//
// It looks trivial and is not: it is the one that would fail if the encoded
// format changed on one side without the other, and that defect would lock ALL
// existing accounts on the next deployment.
//
// The cases are not decorative. The encoded format is
// `$argon2id$v=…$m=…,t=…,p=…$salt$digest`: it uses `$` as a separator. A password
// that contains `$` is therefore the case where a naive implementation — one
// that split before decoding, or that concatenated without escaping — would
// break. Likewise, a non-ASCII password checks that hashing operates on BYTES
// and not on runes: counting in runes would only fail in production, for the
// sole users whose language is written with accents.
func TestHashThenVerifyAcceptsTheRightPassword(t *testing.T) {
	t.Parallel()

	secrets := map[string]string{
		"ASCII phrase":           "correct horse battery staple",
		"accents and spaces":     "a naïve café in Zürich, Straße 12",
		"contains the separator": "pass$word$argon2id$v=19",
		"symbols and digits":     "9!£{}[]|\\~`^@#%&*()_+-=<>?/",
		"ideograms":              "正しい馬のバッテリーステープル",
	}

	hasher := newHasher()
	for name, secret := range secrets {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			encoded := hash(t, secret)

			ok, err := hasher.Verify(secret, encoded)
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if !ok {
				t.Error("the right password must be accepted")
			}
		})
	}
}
