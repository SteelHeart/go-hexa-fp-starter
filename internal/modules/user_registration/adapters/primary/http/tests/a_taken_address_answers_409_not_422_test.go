package tests

import (
	"net/http"
	"testing"
)

// TestATakenAddressAnswers409Not422: an already taken address returns 409, not
// 422.
//
// # The distinction this test protects
//
// 422 says "your request is malformed". 409 says "your request is perfectly well
// formed, it is the state of the server that opposes it". These are not two ways
// of saying the same thing:
//
//   - A registration form must display "this address is already in use, please
//     sign in" on a 409, and "invalid address" on a 422. Confusing them produces
//     the most frustrating message there is: telling someone their address is
//     invalid when it is simply theirs.
//   - An automated client may treat the 409 as an idempotent success; it can
//     never do that with a 422.
//
// The test registers first, then replays with a DIFFERENT case: it is the domain
// that normalises, so `ALICE@EXAMPLE.COM` must collide with
// `alice@example.com`. Without normalisation, two accounts would be created for
// one and the same person — and the duplicate would only be seen by support.
func TestATakenAddressAnswers409Not422(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	first := post(t, server, registerBody(t, "alice@example.com", validPassword))
	if first.status != http.StatusCreated {
		t.Fatalf("first registration: status = %d, want 201", first.status)
	}

	resp := post(t, server, registerBody(t, "  ALICE@Example.com ", validPassword))
	if resp.status != http.StatusConflict {
		t.Fatalf("status = %d, want 409 — the normalised address is already taken", resp.status)
	}
	if got := resp.body["detail"]; got != "cette adresse de courriel est déjà enregistrée" {
		t.Errorf("detail = %v, want the message of the domain", got)
	}
	assertFieldLocated(t, resp.body, "body.email")
}
