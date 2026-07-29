package tests

import (
	"net/http"
	"testing"
)

// TestEachDomainErrorMapsToItsOwnStatus: every refusal of the domain produces
// ITS status, and the returned message comes from the domain.
//
// # Why distinguishing the statuses is not cosmetic
//
// An automated client decides whether or not to retry from the status:
//
//   - **422** — the request is malformed. Retrying identically will always
//     fail. The client has to correct, or give up.
//   - **409** — the request is WELL formed, it is the state of the server that
//     opposes it. The client knows it is not its fault.
//   - **503** — transient. Retrying later makes sense.
//
// Returning everything as a 400, or worse as a 500, forces every client to parse
// a message in French in order to decide. And a 500 on a typing mistake wakes
// somebody up at night for a user who mistyped their address.
//
// The test also verifies that the message comes from the DOMAIN. A validation
// constraint laid down in the HTTP schema would short circuit the domain and
// return the message of the framework — in English, in the middle of a
// French-speaking API. That defect really existed here before being removed.
func TestEachDomainErrorMapsToItsOwnStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		email, password string
		wantStatus      int
		wantDetail      string
		wantField       string
	}{
		"invalid address": {
			email: "not-an-address", password: validPassword,
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "l'adresse de courriel n'est pas valide",
			wantField:  "body.email",
		},
		"password too short": {
			email: "bob@example.com", password: "short",
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "le mot de passe doit faire au moins 12 caractères",
			wantField:  "body.password",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newServer(t)
			resp := post(t, server, registerBody(t, tc.email, tc.password))

			if resp.status != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.status, tc.wantStatus)
			}
			if got := resp.body["detail"]; got != tc.wantDetail {
				t.Errorf("detail = %v, want the message of the domain %q", got, tc.wantDetail)
			}
			assertFieldLocated(t, resp.body, tc.wantField)
		})
	}
}

// assertFieldLocated verifies that the response names the faulty field.
//
// Without it, a form does not know what to highlight and displays the error at
// the top of the page — which, on a two-field form, forces the user to guess
// which one to redo.
func assertFieldLocated(t *testing.T, body map[string]any, want string) {
	t.Helper()

	errs, ok := body["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Fatalf("the response carries no error detail: %v", body)
	}
	first, ok := errs[0].(map[string]any)
	if !ok {
		t.Fatalf("unreadable error detail: %v", errs[0])
	}
	if got := first["location"]; got != want {
		t.Errorf("location = %v, want %q — the faulty field must be named", got, want)
	}
}
