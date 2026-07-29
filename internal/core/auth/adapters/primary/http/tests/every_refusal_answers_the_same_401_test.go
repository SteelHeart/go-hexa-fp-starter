package tests

import (
	"net/http"
	"testing"
)

// TestEveryRefusalAnswersTheSame401 closes the oracle AT THE SURFACE.
//
// # Why replay it here when the use case already guarantees it
//
// Because the surface has its own opportunity to break it. The module returns a
// single sentinel, but it is the translator that chooses the status AND the
// message — and a single `err.Error()` returned as is, or a well-meaning
// "unknown account" 404, is enough to reopen what the domain had closed.
//
// The test compares the responses AGAINST EACH OTHER, not against an expected
// value: indistinguishability is the property.
func TestEveryRefusalAnswersTheSame401(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	cases := map[string][2]string{
		"unknown subject":   {"nobody@example.com", secret},
		"wrong secret":      {subject, "another-secret-quite-long-enough"},
		"empty subject":     {"", secret},
		"empty secret":      {subject, ""},
		"malformed subject": {"   ", secret},
	}

	bodies := make(map[string]bool)
	for name, tc := range cases {
		resp := openSession(t, server, tc[0], tc[1])
		if resp.status != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d — body %s", name, resp.status, resp.raw)
		}
		bodies[resp.raw] = true
	}

	if len(bodies) != 1 {
		t.Fatalf("the refusals must be indistinguishable; %d distinct bodies: %v", len(bodies), bodies)
	}
}

// TestAMalformedBearerNeverSaysWhy applies the same rule to the token.
//
// Missing header, unknown scheme, token too short, invented token: a single
// status and a single body. Distinguishing "malformed" from "unknown" would
// tell an attacker that their string has the right SHAPE — hence that they are
// getting close.
func TestAMalformedBearerNeverSaysWhy(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	headers := map[string]string{
		"missing":              "",
		"missing scheme":       "0123456789012345678901234567890123456789012",
		"unknown scheme":       "Basic 0123456789012345678901234567890123456789012",
		"token too short":      "Bearer too-short",
		"empty token":          "Bearer ",
		"invented token":       "Bearer 9999999999999999999999999999999999999999999",
		"scheme without token": "Bearer",
	}

	bodies := make(map[string]bool)
	for name, header := range headers {
		resp := withBearer(t, server, http.MethodGet, identityPath, header)
		if resp.status != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d — body %s", name, resp.status, resp.raw)
		}
		bodies[resp.raw] = true
	}

	if len(bodies) != 1 {
		t.Fatalf("the token refusals must be indistinguishable; %d bodies: %v", len(bodies), bodies)
	}
}

// TestTheBearerSchemeIsCaseInsensitive follows RFC 7235.
//
// The authentication scheme is declared case-insensitive there. Refusing
// `bearer ` would fail perfectly correct clients, with a 401 that would blame
// the token — hence sending people to look for the fault in the wrong place.
func TestTheBearerSchemeIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	for _, scheme := range []string{"Bearer ", "bearer ", "BEARER ", "BeArEr "} {
		resp := withBearer(t, server, http.MethodGet, identityPath, scheme+token)
		if resp.status != http.StatusOK {
			t.Errorf("scheme %q: want 200, got %d — body %s", scheme, resp.status, resp.raw)
		}
	}
}
