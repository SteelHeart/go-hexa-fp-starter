package tests

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestMalformedHashIsRefused: an unreadable digest returns an ERROR, not a
// refusal.
//
// A digest is supposed to come from our own database — but a data import, a
// truncated column or an external store are enough to slip something else in.
// Treating it as "wrong password" would hide a data corruption behind login
// failures that nobody would ever connect to one another.
func TestMalformedHashIsRefused(t *testing.T) {
	t.Parallel()

	valid := hash(t, "correct horse battery staple")
	parts := strings.Split(valid, "$")

	cases := map[string]string{
		"empty":                 "",
		"no structure":          "not a digest",
		"unknown algorithm":     strings.Replace(valid, "argon2id", "bcrypt", 1),
		"unexpected version":    strings.Replace(valid, "v=19", "v=13", 1),
		"unreadable parameters": "$argon2id$v=19$memory=lots$" + parts[4] + "$" + parts[5],
		"salt not base64":       "$argon2id$v=19$" + parts[3] + "$!!!$" + parts[5],
		"key not base64":        "$argon2id$v=19$" + parts[3] + "$" + parts[4] + "$!!!",
		"missing segment":       strings.Join(parts[:5], "$"),
	}

	hasher := newHasher()
	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ok, err := hasher.Verify("whatever", encoded)
			if !errors.Is(err, security.ErrInvalidHash) {
				t.Errorf("error = %v, want ErrInvalidHash", err)
			}
			if ok {
				t.Error("an unreadable digest must NEVER validate a password")
			}
		})
	}

	// A digest announcing an absurd key is refused BEFORE reaching argon2.
	// Without this bound, the announced length directly drives an allocation:
	// a forged "digest" would bring the process down on the first verification.
	oversized := "$argon2id$v=19$" + parts[3] + "$" + parts[4] + "$" +
		base64.RawStdEncoding.EncodeToString(make([]byte, 1<<20))
	if ok, err := hasher.Verify("whatever", oversized); !errors.Is(err, security.ErrInvalidHash) || ok {
		t.Errorf("an outsized key must be refused: ok=%v err=%v", ok, err)
	}
}
