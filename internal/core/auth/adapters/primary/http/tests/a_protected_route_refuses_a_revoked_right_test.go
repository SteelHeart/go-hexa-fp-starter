package tests

import (
	"net/http"
	"testing"
)

// TestAProtectedRouteRefusesARevokedRight carries ADR 017 ALL THE WAY TO THE
// SURFACE.
//
// # Why replay it here when the module already guarantees it
//
// Because the surface has its own opportunity to break it. The module's witness
// proves that `Authorize` queries the persisted state; this one proves that the
// route CALLS it — on every request, and not once at mount time.
//
// The mistake it catches is mundane and invisible: resolving the permission at
// startup then memoising the decision, or adding a middleware that caches the
// resolved identity. Both pass every test of the module, and both keep a
// revoked right active.
//
// The token stays valid throughout — the test checks that explicitly between
// the two calls — otherwise the refusal could come from an invalidated session
// and the demonstration would be empty.
func TestAProtectedRouteRefusesARevokedRight(t *testing.T) {
	t.Parallel()

	server, token := bootstrappedServer(t)

	// The bootstrap account holds the admin role: the route answers.
	body := `{"subject":"bob@example.com","secret":"correct horse battery staple"}`
	if resp := send(t, server, request{http.MethodPost, identitiesPath, bearerOf(token), body}); resp.status != http.StatusCreated {
		t.Fatalf("the bootstrap account must be able to create: want 201, got %d — %s",
			resp.status, resp.raw)
	}

	// Revocation of the RIGHT, not of the session: the role is redefined without
	// the creation permission. No token is touched.
	empty := `{"permissions":[]}`
	if resp := send(t, server, request{http.MethodPut, rolePath, bearerOf(token), empty}); resp.status != http.StatusNoContent {
		t.Fatalf("redefining the role: want 204, got %d — %s", resp.status, resp.raw)
	}

	// The token is STILL worth something: that is what makes the refusal that
	// follows conclusive.
	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusOK {
		t.Fatalf("the token should not have been affected by the revocation: %d — %s",
			resp.status, resp.raw)
	}

	other := `{"subject":"carol@example.com","secret":"correct horse battery staple"}`
	resp := send(t, server, request{http.MethodPost, identitiesPath, bearerOf(token), other})
	if resp.status != http.StatusForbidden {
		t.Fatalf("revoked right: want 403, got %d — %s", resp.status, resp.raw)
	}
}

// TestAnUnauthenticatedCallerNeverLearnsThatARouteIsGuarded guards the order of
// the refusals.
//
// # The order of the refusals is the decision
//
// A missing or invalid token returns **401**, never 403. Saying "permission
// denied" to someone who is not authenticated would teach them two things: that
// the route exists, and that a right guards it. That is exactly the map an
// attacker builds before acting.
//
// The 403 is reserved for an AUTHENTICATED bearer — and there it is useful: it
// spares them signing in over and over for a right they will not have any more
// than before.
func TestAnUnauthenticatedCallerNeverLearnsThatARouteIsGuarded(t *testing.T) {
	t.Parallel()

	server, _ := bootstrappedServer(t)
	body := `{"subject":"bob@example.com","secret":"correct horse battery staple"}`

	headers := map[string]string{
		"missing":         "",
		"unknown scheme":  "Basic " + unknownToken,
		"invented token":  "Bearer " + unknownToken,
		"token too short": "Bearer short",
	}
	for name, header := range headers {
		resp := send(t, server, request{http.MethodPost, identitiesPath, header, body})
		if resp.status != http.StatusUnauthorized {
			t.Errorf("%s: want 401, got %d — %s", name, resp.status, resp.raw)
		}
	}
}

// TestAnAuthenticatedCallerWithoutTheRightGets403 guards the other half.
//
// The account created by the administrator has NO role: it authenticates
// perfectly and obtains nothing. That is deny by default seen from the surface,
// and it is the half the previous test does not cover — a guard that returned
// 401 to everybody would be green there and perfectly useless.
func TestAnAuthenticatedCallerWithoutTheRightGets403(t *testing.T) {
	t.Parallel()

	server, admin := bootstrappedServer(t)

	const subj = "bob@example.com"
	body := `{"subject":"` + subj + `","secret":"` + secret + `"}`
	if resp := send(t, server, request{http.MethodPost, identitiesPath, bearerOf(admin), body}); resp.status != http.StatusCreated {
		t.Fatalf("creating the account: %d — %s", resp.status, resp.raw)
	}

	withoutRight := tokenOf(t, openSession(t, server, subj, secret))
	resp := send(t, server, request{
		http.MethodPost, identitiesPath, bearerOf(withoutRight),
		`{"subject":"carol@example.com","secret":"` + secret + `"}`,
	})
	if resp.status != http.StatusForbidden {
		t.Fatalf("authenticated without the right: want 403, got %d — %s", resp.status, resp.raw)
	}
}

// TestClosingAnAccountTakesEffectAtOnce: the closure holds on the next call.
//
// This is the gesture you make when an account is compromised, hence the only
// moment where latency really matters. The closed account's token was issued
// BEFORE the closure — it is the one an attacker holds at the moment you
// react.
func TestClosingAnAccountTakesEffectAtOnce(t *testing.T) {
	t.Parallel()

	server, admin := bootstrappedServer(t)

	const subj = "bob@example.com"
	creation := send(t, server, request{
		http.MethodPost, identitiesPath, bearerOf(admin),
		`{"subject":"` + subj + `","secret":"` + secret + `"}`,
	})
	if creation.status != http.StatusCreated {
		t.Fatalf("creation: %d — %s", creation.status, creation.raw)
	}
	id, _ := creation.body["identity_id"].(string)

	token := tokenOf(t, openSession(t, server, subj, secret))
	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusOK {
		t.Fatalf("the token has just been issued: %d", resp.status)
	}

	if resp := send(t, server, request{http.MethodDelete, identitiesPath + "/" + id, bearerOf(admin), ""}); resp.status != http.StatusNoContent {
		t.Fatalf("closure: want 204, got %d — %s", resp.status, resp.raw)
	}

	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusUnauthorized {
		t.Fatalf("closed account, token already issued: want 401, got %d — %s", resp.status, resp.raw)
	}
}
