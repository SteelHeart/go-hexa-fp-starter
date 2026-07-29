// Package tests exercises the HTTP middlewares as a BLACK BOX.
//
// These are the guards EVERY request goes through: hardening headers, CORS, rate
// limiting, body bound, panic recovery. None of them was covered.
//
// An untested security middleware is the worst place to be untested: it produces
// no symptom when it stops protecting. An `Access-Control-Allow-Origin` granted
// by mistake breaks nothing, shows up in no log, and is only discovered the day
// someone uses it.
package tests

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// okHandler answers 200 and reports whether it was reached.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if reached != nil {
			*reached = true
		}
		w.WriteHeader(http.StatusOK)
	})
}

// call sends a request through a middleware and returns the response.
func call(mw func(http.Handler) http.Handler, req *http.Request, next http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec
}

// get builds a GET request to /.
func get(t *testing.T) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
}

// post builds a POST request carrying a body.
func post(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
}

// preflight builds a CORS preflight request.
func preflight(t *testing.T, origin string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/", http.NoBody)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	return req
}

// discardLogger throws logs away.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
