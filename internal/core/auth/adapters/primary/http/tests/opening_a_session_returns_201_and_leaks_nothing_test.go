package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestOpeningASessionReturns201AndLeaksNothing exercises the happy path AND the
// discretion of the response.
//
// # What the RAW body adds
//
// A field-by-field inspection would miss a secret carried under an innocuous
// name. Searching in the text, on the other hand, depends on no assumption
// about the shape of the leak.
func TestOpeningASessionReturns201AndLeaksNothing(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	resp := openSession(t, server, subject, secret)

	if resp.status != http.StatusCreated {
		t.Fatalf("want 201, got %d — body %s", resp.status, resp.raw)
	}

	token := tokenOf(t, resp)
	if len(token) < 43 {
		t.Fatalf("the token must carry at least 256 bits of randomness, got %d characters", len(token))
	}
	if _, ok := resp.body["identity_id"].(string); !ok {
		t.Fatalf("the response must carry the identifier: %s", resp.raw)
	}
	if _, ok := resp.body["expires_at"].(string); !ok {
		t.Fatalf("the response must bound the session in time: %s", resp.raw)
	}

	if strings.Contains(resp.raw, secret) {
		t.Fatalf("the PLAIN secret leaks in the response: %s", resp.raw)
	}
	if strings.Contains(resp.raw, "digest") {
		t.Fatalf("the digest leaks in the response: %s", resp.raw)
	}

	// No permission, no role: the token authenticates, it does not authorise.
	// Publishing them there would invite every client to cache them, and
	// revocation would stop being immediate without a single line of this
	// repository changing.
	for _, forbidden := range []string{"permission", "permissions", "roles", "scope", "scopes"} {
		if _, present := resp.body[forbidden]; present {
			t.Fatalf("the sign-in response carries %q: %s", forbidden, resp.raw)
		}
	}
}

// TestTwoSessionsGetTwoDistinctTokens guards the token's randomness.
//
// Two sign-ins of the same account must produce two different tokens. A token
// derived from the subject — or worse, constant — would be guessable, and a
// guessable authentication is not one.
func TestTwoSessionsGetTwoDistinctTokens(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	first := tokenOf(t, openSession(t, server, subject, secret))
	second := tokenOf(t, openSession(t, server, subject, secret))

	if first == second {
		t.Fatal("two sign-ins produced the same token")
	}
	if strings.Contains(first, subject) {
		t.Fatalf("the token is derived from the subject: %q", first)
	}
}
