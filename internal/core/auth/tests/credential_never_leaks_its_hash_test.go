package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestCredentialNeverLeaksItsHash covers BOTH formatting verbs.
//
// # Why two, and not one
//
// `%v` goes through `String()`, `%#v` through `GoString()`. Covering one leaves
// the other leaking, and `%#v` is precisely what one writes in a debug log —
// hence on the day of an incident, hence on the day the logs go to a third
// party.
//
// An Argon2id digest is not a password, but it can be cracked offline:
// publishing it turns a log leak into an account leak.
//
// ⚠️ The test DELIBERATELY formats with `%v` and `%#v` rather than calling
// `String()`: it is the real leak path that is exercised. Calling the method
// would leave the test green if someone removed the `Stringer`.
func TestCredentialNeverLeaksItsHash(t *testing.T) {
	t.Parallel()

	const hash = "$argon2id$v=19$m=65536,t=3,p=4$c2VsLWZhY3RpY2U$Y29uZGVuc2UtZmFjdGljZQ"

	subj, err := domain.NewSubject(subject)
	if err != nil {
		t.Fatalf("subject: %v", err)
	}
	identity, err := domain.NewIdentity("id-1", subj, nil, time.Now())
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	credential, err := domain.NewCredential(identity, hash)
	if err != nil {
		t.Fatalf("credential: %v", err)
	}

	for _, rendered := range []string{
		fmt.Sprintf("%v", credential),  //nolint:gocritic // this is the leak path under test
		fmt.Sprintf("%#v", credential), // through GoString — the other half of the mask
		fmt.Sprintf("%s", credential),  //nolint:gocritic,staticcheck // idem, through String
		fmt.Sprint(credential),         //nolint:gocritic // idem, without a verb
	} {
		if strings.Contains(rendered, hash) {
			t.Fatalf("the digest leaks in %q", rendered)
		}
		if !strings.Contains(rendered, "***") {
			t.Fatalf("the mask must be visible in %q", rendered)
		}
	}

	// The digest stays ACCESSIBLE, through a named accessor: an access is then
	// visible on review, and can be searched for in a single command.
	if credential.SecretHash() != hash {
		t.Fatal("the digest must stay accessible for comparison")
	}
}

// TestCredentialRefusesAnIncompleteAssembly guards the type's bounds.
//
// Two `string` in a row — "the subject" and "the digest" — get silently
// swapped, and the swap would produce a comparison that always succeeds. The
// type exists to make the swap impossible; the refusal guards its bounds.
func TestCredentialRefusesAnIncompleteAssembly(t *testing.T) {
	t.Parallel()

	subj, err := domain.NewSubject(subject)
	if err != nil {
		t.Fatalf("subject: %v", err)
	}
	identity, err := domain.NewIdentity("id-1", subj, nil, time.Now())
	if err != nil {
		t.Fatalf("identity: %v", err)
	}

	if _, err := domain.NewCredential(domain.Identity{}, "digest"); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("without an identity: want ErrIncomplete, got %v", err)
	}
	if _, err := domain.NewCredential(identity, ""); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("without a digest: want ErrIncomplete, got %v", err)
	}
}

// TestTokenComparesInConstantTime exercises the property through its CONTRACT.
//
// Constant time is not measured honestly in a unit test — a scheduler, a
// garbage collector or a shared machine add more noise to it than the channel
// under study produces signal. What the test guards is the CORRECTNESS of
// `Equals`: the day someone replaced `subtle.ConstantTimeCompare` with `==`
// thinking they were simplifying, review is the only net — and a test that
// requires `Equals` to exist at least prevents the method from disappearing in
// favour of a direct comparison at the caller.
func TestTokenComparesInConstantTime(t *testing.T) {
	t.Parallel()

	const raw = "0123456789012345678901234567890123456789012"

	token, err := domain.NewToken(raw)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	same, err := domain.NewToken(raw)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	other, err := domain.NewToken(strings.Repeat("a", 43))
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	if !token.Equals(same) {
		t.Error("two identical tokens must be equal")
	}
	if token.Equals(other) {
		t.Error("two different tokens must not be equal")
	}
	if token.Equals(domain.Token{}) {
		t.Error("a token that was never built must equal no token")
	}

	// A token that is too short is refused AT CONSTRUCTION: the bound lives in
	// the domain, so it holds whatever port produced the string.
	if _, err := domain.NewToken(strings.Repeat("a", 42)); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("42-character token: want ErrIncomplete, got %v", err)
	}
}
