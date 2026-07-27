package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAnHTTPCallCarriesThePayloadAndDecodesTheReply : l'appel distant est complet.
//
// # Ce que ce test verrouille, et pourquoi chaque point compte
//
//   - La MÉTHODE et le CHEMIN viennent de la Route. Une route ignorée enverrait
//     un GET là où le module distant attend un POST : 405, sur toutes les
//     capacités d'un coup, et seulement après déploiement séparé.
//   - Le corps est la requête sérialisée. C'est la seule chose que le module
//     distant reçoit.
//   - `Content-Type: application/json` est posé. Sans lui, un serveur strict —
//     et huma l'est — refuse le corps avant de l'avoir lu.
//   - L'adresse de base est jointe au chemin SANS barre oblique doublée. Une
//     adresse terminée par `/` est ce que produit n'importe quelle configuration
//     écrite à la main ; `//v1/things` n'est pas la même route pour un routeur.
//   - La réponse est décodée dans le type de sortie. Un appel qui réussirait en
//     rendant la valeur zéro serait le pire des deux mondes.
func TestAnHTTPCallCarriesThePayloadAndDecodesTheReply(t *testing.T) {
	t.Parallel()

	var (
		gotMethod      string
		gotPath        string
		gotContentType string
		gotBody        request
	)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply{Accepted: true})
	}))
	defer remote.Close()

	var localCalls int
	// Adresse terminée par une barre oblique, volontairement.
	cfg := interop("http", map[string]string{someModule: remote.URL + "/"})
	call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-42"})
	if err != nil {
		t.Fatalf("appel distant en échec: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("méthode reçue = %q, attendu POST — la Route n'est pas respectée", gotMethod)
	}
	if gotPath != "/v1/things" {
		t.Errorf("chemin reçu = %q, attendu /v1/things — barre oblique doublée ?", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type reçu = %q — un serveur strict refuserait le corps", gotContentType)
	}
	if gotBody.Ref != "r-42" {
		t.Errorf("corps reçu = %+v, la requête n'a pas traversé", gotBody)
	}
	if !got.Accepted {
		t.Error("la réponse distante n'a pas été décodée — la valeur zéro a été rendue")
	}
	if localCalls != 0 {
		t.Errorf("l'implémentation locale a été appelée %d fois en mode http", localCalls)
	}
}
