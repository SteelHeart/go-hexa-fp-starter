package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEachAccountGetsItsOwnHash: the REAL password is hashed, not its masked
// form.
//
// # The defect this test catches
//
// `RawPassword.String()` returns "[redacted]" — that is the protection which
// prevents accidental logging from leaking a password. A hashing adapter that
// calls `String()` instead of `Expose()` compiles, passes review, and hashes the
// SAME string for every account.
//
// Consequence: all users share one digest, and any password opens any account.
// The breach is total and perfectly invisible — registration succeeds, sign-in
// succeeds, nothing looks abnormal.
//
// This defect was actually written during the development of this adapter. This
// test exists so that it cannot come back.
func TestEachAccountGetsItsOwnHash(t *testing.T) {
	t.Parallel()

	// Minimal cost: what is exercised is the WIRING, not the robustness of
	// Argon2, which has its own tests in internal/infrastructure/security.
	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  1 << 10,
		Iterations: 1,
		Threads:    1,
	})
	hashPort := hashing.New(hasher)

	publisher := &spyPublisher{}
	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: hashPort,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err != nil {
		t.Fatalf("mounting the module: %v", err)
	}

	first := register(t, mod, "alice@example.com", "correct horse battery staple")
	second := register(t, mod, "bob@example.com", "an altogether different secret phrase")

	assertRealPasswordWasHashed(t, hasher, first, "correct horse battery staple")
	assertRealPasswordWasHashed(t, hasher, second, "an altogether different secret phrase")

	if first.PasswordHash.String() == second.PasswordHash.String() {
		t.Fatal("two accounts share the same digest — the masked value was hashed instead of the password")
	}
}

// assertRealPasswordWasHashed verifies that the digest matches the password
// actually supplied.
func assertRealPasswordWasHashed(
	t *testing.T,
	hasher security.Hasher,
	user domain.User,
	plain string,
) {
	t.Helper()

	ok, err := hasher.Verify(plain, user.PasswordHash.String())
	if err != nil {
		t.Fatalf("verifying the digest of %s: %v", user.Email, err)
	}
	if !ok {
		t.Errorf("the digest of %s does not match the supplied password", user.Email)
	}
	if verified, _ := hasher.Verify("[redacted]", user.PasswordHash.String()); verified {
		t.Errorf("the digest of %s matches \"[redacted]\": it is the MASKED value that was hashed", user.Email)
	}
}
