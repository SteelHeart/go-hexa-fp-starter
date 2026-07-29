package domain

import "fmt"

// Credential carries an identity AND the digest of its secret.
//
// # Why this type exists rather than two return values
//
// The architecture rule bounds core functions to TWO return values, and a port
// that returned `(Identity, string, error)` would be refused. It is right here
// for a reason specific to the subject matter: two `string` in a row — "the
// subject" and "the digest" — get silently swapped, and the swap would produce
// a comparison that always succeeds.
//
// # This type is never logged
//
// It implements `Stringer` and `GoStringer` in order to MASK the digest. An
// Argon2id digest is not a password, but it can be cracked offline: publishing
// it in a log turns a log leak into an account leak.
//
// Both implementations are necessary, not redundant: `%v` goes through
// `String()`, `%#v` through `GoString()`. It took a test to discover that
// covering one left the other leaking.
type Credential struct {
	identity   Identity
	secretHash string
}

// NewCredential assembles an identity and the digest of its secret.
func NewCredential(identity Identity, secretHash string) (Credential, error) {
	if identity.ID == "" {
		return Credential{}, fmt.Errorf("%w: the identity is mandatory", ErrIncomplete)
	}
	if secretHash == "" {
		return Credential{}, fmt.Errorf("%w: the secret digest is mandatory", ErrIncomplete)
	}
	return Credential{identity: identity, secretHash: secretHash}, nil
}

// Identity returns the carried identity.
func (c Credential) Identity() Identity { return c.identity }

// SecretHash returns the digest, FOR COMPARISON only.
//
// Named explicitly rather than exposed as a field: an access to the digest is
// then visible on review, and can be searched for in a single command.
func (c Credential) SecretHash() string { return c.secretHash }

// String masks the digest. See the type documentation.
func (c Credential) String() string {
	return "Credential{identity: " + string(c.identity.ID) + ", secretHash: ***}"
}

// GoString masks the digest under `%#v` too.
func (c Credential) GoString() string { return c.String() }
