// Package tests exercises the HTTP SURFACE of the authentication module.
//
// # In process, without any infrastructure
//
// The server is mounted in memory by `httptest`: no port opened, no binary
// launched, no external service. These tests therefore run in an ordinary
// `go test ./...`, on any machine, in a few milliseconds — and they break the
// second someone changes a status code.
//
// What they check: the TRANSLATION, and the leaks. The domain and the use cases
// have their own tests; here we check that a refusal becomes the right status,
// that a secret never comes back out, and that two different refusals stay
// indistinguishable once translated.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	authhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/adapters/primary/http"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

const (
	// subject designates the account used by the tests.
	subject = "alice@example.com"

	// secret satisfies the module's bounds.
	secret = "correct horse battery staple"

	// sessionsPath and identityPath deliberately duplicate the contract's
	// constants: a test that imported them would stay green if someone renamed
	// a route, and a path change is breaking for every already deployed
	// client.
	sessionsPath   = "/v1/auth/sessions"
	currentPath    = "/v1/auth/sessions/current"
	identityPath   = "/v1/auth/identity"
	identitiesPath = "/v1/auth/identities"
	rolePath       = "/v1/auth/roles/admin"

	// unknownToken has the RIGHT shape and was never issued: 43 characters,
	// exactly what the domain demands. That is what makes it possible to
	// exercise the refusal of a well-formed but unknown token.
	unknownToken = "0000000000000000000000000000000000000000000"
)

// hashSecret is a DUMMY hash, instantaneous.
//
// Argon2id is deliberately slow. Paying for it here would make every test last
// tens of milliseconds without exercising anything more: what is tested is the
// HTTP translation, not the strength of the digest.
func hashSecret(plain string) (string, error) { return "digest:" + plain, nil }

func verifySecret(plain, encoded string) (bool, error) { return encoded == "digest:"+plain, nil }

// newServer mounts the surface on an in-memory server, with one registered
// account.
//
// The module runs on its default driver: each test therefore has its own store,
// pristine, and can run in parallel without interfering.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	mod := newModule(t, config.Module{Enabled: true, Driver: "memory"})
	if _, err := mod.Register(context.Background(), subject, secret); err != nil {
		t.Fatalf("registration: %v", err)
	}
	return serve(t, mod)
}

// newModule builds the module or fails the test.
func newModule(t *testing.T, cfg config.Module) auth.Module {
	t.Helper()

	mod, err := auth.New(cfg, auth.Deps{
		HashSecret:   hashSecret,
		VerifySecret: verifySecret,
		Now:          func() time.Time { return time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("mounting the module: %v", err)
	}
	return mod
}

// serve mounts a module on an in-memory server.
func serve(t *testing.T, mod auth.Module) *httptest.Server {
	t.Helper()

	router := httpserver.NewRouter(testConfig(), discardLogger(), nil)
	authhttp.Mount(router.API, mod)

	server := httptest.NewServer(router.Mux)
	t.Cleanup(server.Close)
	return server
}

// testConfig carries the minimum the router needs.
//
// ⚠️ The rate limits are deliberately HIGH: at zero, the limiter would refuse
// everything and every test would fail with a 429 — a defect whose cause would
// be invisible in the error message.
func testConfig() config.Config {
	return config.Config{
		App:    config.App{Env: config.EnvTest, Name: "surface-auth-test", Version: "test"},
		HTTP:   config.HTTP{MaxBodyBytes: 1 << 20},
		Limits: config.Limits{RPS: 10_000, Burst: 10_000},
	}
}

// discardLogger silences the router's logs during the tests.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// response is a fully consumed HTTP response.
//
// The RAW body is kept alongside the decoded one, and it matters just as much:
// a leak is looked for in the TEXT, not in a map. A secret carried by a field
// with an innocuous name would escape any field-by-field inspection.
type response struct {
	status int
	body   map[string]any
	raw    string
}

// openSession asks for a session and returns the response, body read and
// closed.
func openSession(t *testing.T, server *httptest.Server, subj, sec string) response {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"subject": subj, "secret": sec})
	if err != nil {
		t.Fatalf("serialising: %v", err)
	}

	// bodyclose does not follow the closing done by `consume`, which does it in
	// a `defer` on every path nonetheless.
	//
	//nolint:noctx,bodyclose // local httptest; the body is read and closed by consume
	resp, err := server.Client().Post(
		server.URL+sessionsPath, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST %s: %v", sessionsPath, err)
	}
	return consume(t, resp)
}

// withBearer sends a request carrying an `Authorization` header.
//
// The header is passed RAW and not built from a token: that is what makes it
// possible to exercise a missing scheme, an unknown scheme, or an unexpected
// case — the cases where the surface decides on its own, without the module.
func withBearer(t *testing.T, server *httptest.Server, method, path, header string) response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, server.URL+path, http.NoBody)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	if header != "" {
		req.Header.Set("Authorization", header)
	}

	//nolint:bodyclose // the body is read and closed by consume
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return consume(t, resp)
}

// consume reads the response in full and closes it.
//
// An empty body is ACCEPTED without error: a 204 has none, and failing the test
// on its absence would force every caller to tell the cases apart.
func consume(t *testing.T, resp *http.Response) response {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading the body: %v", err)
	}

	var payload map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("unreadable response body (%q): %v", raw, err)
		}
	}
	return response{status: resp.StatusCode, body: payload, raw: string(raw)}
}

// bootstrappedServer mounts the surface WITH an administrator bootstrap
// account.
//
// It is the only way to exercise a protected route: without a role, everything
// returns 403, and a test that only ever saw 403s would not tell a correct
// guard from a guard that refuses everything.
func bootstrappedServer(t *testing.T) (server *httptest.Server, adminToken string) {
	t.Helper()

	mod := newModule(t, config.Module{Enabled: true, Driver: "memory"})
	report, err := auth.Bootstrap(context.Background(), mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("bootstrapping: %v", err)
	}
	if !report.Created {
		t.Fatal("bootstrapping should have created the administrator account")
	}

	server = serve(t, mod)
	return server, tokenOf(t, openSession(t, server, report.Subject, report.Secret))
}

// request describes a complete HTTP call.
//
// A structure rather than six parameters: the shape rule allows five, and above
// all `header` and `body` are two neighbouring strings — swapping them
// compiles, and produces a request without a body carrying a JSON document as
// its authorisation. Naming the fields makes the swap visible.
type request struct {
	method string
	path   string
	header string
	body   string
}

// bearerOf builds the authorisation header of a token.
func bearerOf(token string) string { return "Bearer " + token }

// send runs the request and returns the response, body read and closed.
func send(t *testing.T, server *httptest.Server, r request) response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(), r.method, server.URL+r.path, strings.NewReader(r.body))
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	if r.body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.header != "" {
		req.Header.Set("Authorization", r.header)
	}

	//nolint:bodyclose // the body is read and closed by consume
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", r.method, r.path, err)
	}
	return consume(t, resp)
}

// tokenOf extracts the token from a sign-in response.
func tokenOf(t *testing.T, resp response) string {
	t.Helper()

	token, ok := resp.body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("the response carries no usable token: %s", resp.raw)
	}
	return token
}
