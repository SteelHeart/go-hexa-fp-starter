package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestARevokedTokenStopsResolving exercises revocation END TO END.
//
// Three requests: open, close, try again. That is the journey a client really
// makes, and it is the one a self-contained signed token could not interrupt
// without a revocation list — hence without falling back on the store it
// claimed to avoid (ADR 017 § 2 bis).
func TestARevokedTokenStopsResolving(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusOK {
		t.Fatalf("the token has just been issued: want 200, got %d — %s", resp.status, resp.raw)
	}

	resp := withBearer(t, server, http.MethodDelete, currentPath, "Bearer "+token)
	if resp.status != http.StatusNoContent {
		t.Fatalf("closure: want 204, got %d — %s", resp.status, resp.raw)
	}
	if strings.TrimSpace(resp.raw) != "" {
		t.Fatalf("a 204 carries no body, got %q", resp.raw)
	}

	after := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token)
	if after.status != http.StatusUnauthorized {
		t.Fatalf("revoked token: want 401, got %d — %s", after.status, after.raw)
	}
}

// TestClosingATwiceClosedSessionStillAnswers204 guards idempotency at the
// surface.
//
// A client who signs out twice has done nothing wrong — a double click is
// enough. Returning 401 on the second call would display an error at the very
// moment the user has just succeeded in leaving.
func TestClosingATwiceClosedSessionStillAnswers204(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	for attempt := 1; attempt <= 2; attempt++ {
		resp := withBearer(t, server, http.MethodDelete, currentPath, "Bearer "+token)
		if resp.status != http.StatusNoContent {
			t.Fatalf("closure no. %d: want 204, got %d — %s", attempt, resp.status, resp.raw)
		}
	}
}

// TestResolvingAnIdentityCarriesNoPermission records what the response does NOT
// carry.
//
// It says WHO presents the token. A client that deduced from it what they are
// allowed to do would be wrong at the first withdrawal of a right — and that is
// precisely the day one does not want to be wrong.
func TestResolvingAnIdentityCarriesNoPermission(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token)
	if resp.status != http.StatusOK {
		t.Fatalf("want 200, got %d — %s", resp.status, resp.raw)
	}
	if got, _ := resp.body["subject"].(string); got != subject {
		t.Fatalf("want subject %q, got %q", subject, got)
	}

	for _, forbidden := range []string{"permission", "permissions", "scope", "scopes", "token"} {
		if _, present := resp.body[forbidden]; present {
			t.Fatalf("the response carries %q: %s", forbidden, resp.raw)
		}
	}
	if strings.Contains(resp.raw, token) {
		t.Fatalf("the token is returned by a read route: %s", resp.raw)
	}
	if strings.Contains(resp.raw, "digest") {
		t.Fatalf("the digest leaks: %s", resp.raw)
	}
}
