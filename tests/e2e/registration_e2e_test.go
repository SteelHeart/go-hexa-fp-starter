//go:build e2e

// Package e2e exercises the DEPLOYED BINARY, over the network.
//
// # What this level adds, and what it does not
//
// The HTTP surface already has its tests, in process, under
// `internal/modules/user_registration/adapters/primary/http/tests/`. They are
// faster, more precise, and run everywhere. Repeating here what they verify
// would bring nothing but a duplicate to maintain.
//
// This level verifies what they CANNOT verify: that the compiled binary, with
// the SHIPPED configuration, really starts and answers. That is, everything
// that happens between `go build` and the first request — configuration
// loading, module mounting, middleware stack, network listening.
//
// This is exactly the class of defect an in-process test lets through: an
// invalid configuration, a module that refuses to mount, a port that does not
// listen.
//
// # Running
//
//	go test -tags=e2e ./tests/e2e/...
//
// ⚠️ WITHOUT the tag, this file is not compiled: `go test ./tests/e2e/...`
// prints `ok` while running NOTHING. Checking the exit code is not enough —
// one has to check that tests really ran.
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// baseURL is the address of the server started by the CI.
func baseURL(t *testing.T) string {
	t.Helper()

	url := os.Getenv("E2E_BASE_URL")
	if url == "" {
		t.Skip("E2E_BASE_URL missing: no server to query")
	}
	return url
}

// TestDeployedBinaryRegistersAUser: the compiled binary answers a registration.
//
// The complete journey, over the network: loading of the shipped
// configuration, module mounting, middleware stack, listening, handling.
func TestDeployedBinaryRegistersAUser(t *testing.T) {
	url := baseURL(t)
	client := &http.Client{Timeout: 10 * time.Second}

	// One address per run: the CI can replay the job without the store, if it
	// is durable, making the second attempt fail with a 409.
	email := "e2e-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}

	resp, err := client.Post(url+"/v1/users", "application/json", bytes.NewReader(body)) //nolint:noctx // client with an explicit timeout
	if err != nil {
		t.Fatalf("POST /v1/users: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("unreadable body: %v", err)
	}
	if got := payload["status"]; got != "pending" {
		t.Errorf("status = %v, want pending", got)
	}
	if id, ok := payload["user_id"].(string); !ok || id == "" {
		t.Errorf("user_id = %v, want a non-empty identifier", payload["user_id"])
	}
}

// TestDeployedBinaryServesItsContract: the OpenAPI contract is served.
//
// A contract the binary does not expose is a contract nobody can consume:
// client SDKs are derived from it. The BARE path `/openapi` returns 404 —
// that is a mistake already made here, and this test locks it down.
func TestDeployedBinaryServesItsContract(t *testing.T) {
	url := baseURL(t)
	client := &http.Client{Timeout: 10 * time.Second}

	for _, path := range []string{"/openapi.json", "/openapi.yaml", "/docs", "/healthz"} {
		resp, err := client.Get(url + path) //nolint:noctx // client with an explicit timeout
		if err != nil {
			t.Errorf("GET %s: %v", path, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, resp.StatusCode)
		}
	}
}
