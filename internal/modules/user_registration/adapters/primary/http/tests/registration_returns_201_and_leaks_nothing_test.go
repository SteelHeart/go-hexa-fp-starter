package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestRegistrationReturns201AndLeaksNothing: the nominal path, and above all
// what the response does NOT contain.
//
// # The two halves of this test
//
// The first is unremarkable: 201, and the expected fields.
//
// The second is not. The body is serialised from the type of the PUBLISHED
// LANGUAGE, not from `domain.User`. The day someone "simplifies" by returning
// the domain entity directly, the password digest goes out in the HTTP response
// — and nothing reports it, because the response remains valid JSON with the
// right fields plus some more.
//
// This test reads the RAW body and refuses to find the digest in it. It is the
// only way to catch a leak through the addition of a field.
func TestRegistrationReturns201AndLeaksNothing(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	resp := post(t, server, registerBody(t, "Alice@Example.COM ", validPassword))

	if resp.status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.status)
	}
	body, raw := resp.body, resp.raw

	// The address is normalised by the DOMAIN — case and whitespace included.
	// The surface performs no normalisation: if it did, the two would diverge
	// one day.
	if got := body["email"]; got != "alice@example.com" {
		t.Errorf("email = %v, want the normalised address alice@example.com", got)
	}
	// An account is born PENDING. Being born active would be a fail-open: the
	// address is not proven yet.
	if got := body["status"]; got != "pending" {
		t.Errorf("status = %v, want pending — an account is never born active", got)
	}
	if got, ok := body["user_id"].(string); !ok || got == "" {
		t.Errorf("user_id = %v, want a non-empty identifier", body["user_id"])
	}

	// Searching in the TEXT of the response, not in the field names: a digest
	// carried by a harmless-looking field would escape any field-by-field
	// inspection.
	forbidden := map[string]string{
		"the password digest":     "hashed:",
		"the clear-text password": validPassword,
	}
	for what, needle := range forbidden {
		if strings.Contains(raw, needle) {
			t.Errorf("%s appears in the HTTP response: %s", what, raw)
		}
	}
	if _, present := body["password_hash"]; present {
		t.Error("the response exposes password_hash — the body must come from the published language, not from the domain")
	}
}
