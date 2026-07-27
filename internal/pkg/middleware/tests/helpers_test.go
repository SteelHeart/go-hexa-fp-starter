// Package tests éprouve les middlewares HTTP en BOÎTE NOIRE.
//
// Ce sont les gardes que traverse CHAQUE requête : en-têtes de durcissement,
// CORS, limitation de débit, borne de corps, récupération de panique. Aucun
// n'était couvert.
//
// Un middleware de sécurité non testé est le pire endroit où l'être : il ne
// produit aucun symptôme quand il cesse de protéger. Un `Access-Control-Allow-Origin`
// accordé par erreur ne casse rien, n'apparaît dans aucun journal, et ne se
// découvre que le jour où quelqu'un s'en sert.
package tests

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// okHandler répond 200 et signale s'il a été atteint.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if reached != nil {
			*reached = true
		}
		w.WriteHeader(http.StatusOK)
	})
}

// call fait passer une requête à travers un middleware et rend la réponse.
func call(mw func(http.Handler) http.Handler, req *http.Request, next http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec
}

// get construit une requête GET vers /.
func get(t *testing.T) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
}

// post construit une requête POST portant un corps.
func post(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
}

// preflight construit une pré-vérification CORS.
func preflight(t *testing.T, origin string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/", http.NoBody)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	return req
}

// discardLogger jette les journaux.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
