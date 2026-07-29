// Package tests exercises the HTTP SURFACE of the registration module.
//
// # In process, with no infrastructure at all
//
// The server is mounted in memory by `httptest`: no port opened, no binary
// launched, no external service. These tests therefore run in an ordinary `go
// test ./...`, on any machine, in a few milliseconds.
//
// That is deliberate. A surface test that requires orchestrating a binary and a
// database only runs in CI; it is therefore written once, broken a month later,
// and nobody sees it before the next integration. This one breaks the second
// someone changes a status code.
//
// What they verify: the TRANSLATION. The domain has its own tests, so does the
// pipeline. Here the only checks are that a domain error becomes the right HTTP
// status, and that no internal field leaks into the response.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	userhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/http"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// validPassword satisfies the bounds of the domain.
const validPassword = "correct horse battery staple"

// newServer mounts the surface on an in-memory server.
//
// The module runs on its default driver: every test therefore has its own store,
// pristine, and can run in parallel without interfering.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: noopPublish,
		GenerateID:   func() domain.UserID { return "019f9b46-3aec-735a-977d-129192ef130f" },
		Now:          func() time.Time { return time.Date(2026, time.July, 26, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("mounting the module: %v", err)
	}

	router := httpserver.NewRouter(testConfig(), discardLogger(), nil)
	userhttp.Mount(router.API, mod)
	userhttp.MountAvailability(router.API, mod)

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
		App: config.App{
			Env:     config.EnvTest,
			Name:    "surface-test",
			Version: "test",
		},
		HTTP:   config.HTTP{MaxBodyBytes: 1 << 20},
		Limits: config.Limits{RPS: 10_000, Burst: 10_000},
	}
}

// response is a fully consumed HTTP response.
//
// The RAW body is kept alongside the decoded one, and it matters just as much: a
// leak is looked for in the TEXT, not in a map. A digest carried by a
// harmless-looking field would escape any field-by-field inspection.
type response struct {
	status int
	body   map[string]any
	raw    string
}

// post sends a registration and returns the response, body read and closed.
//
// The helper does NOT let a `*http.Response` escape, deliberately: a body left
// open holds a connection, and entrusting its closing to every test is the
// guarantee that one test will forget it. Here, the question no longer arises.
func post(t *testing.T, server *httptest.Server, body string) response {
	t.Helper()

	// bodyclose does not follow the closing done by `consume`, which
	// nonetheless does it in a `defer` on every path. The motive is written
	// down rather than the directive laid down without a reason.
	//
	//nolint:noctx,bodyclose // local httptest; the body is read and closed by consume
	resp, err := server.Client().Post(
		server.URL+"/v1/users", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /v1/users: %v", err)
	}
	return consume(t, resp)
}

// consume reads the response in full and closes it.
func consume(t *testing.T, resp *http.Response) response {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading the body: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unreadable response body (%q): %v", raw, err)
	}
	return response{status: resp.StatusCode, body: payload, raw: string(raw)}
}

// registerBody builds a registration payload.
func registerBody(t *testing.T, email, password string) string {
	t.Helper()

	raw, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("serialising the request: %v", err)
	}
	return string(raw)
}

// fakeHash avoids the cost of Argon2: it is not the cryptography that is tested
// here, it has its own tests.
func fakeHash(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("hashed:" + password.Expose()),
	)
}

// noopPublish accepts any event without doing anything with it.
//
// Publication has its own tests, in the module and in `relay`: here it only has
// to succeed, so that the nominal path reaches the HTTP response.
func noopPublish(
	_ context.Context, _, _ string, _ any,
) result.Result[domain.Ack, domain.Error] {
	return result.Ok[domain.Ack, domain.Error](domain.Ack{})
}

// discardLogger throws logs away: a test that prints them drowns its own output,
// and it is not the logging that is verified here.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
