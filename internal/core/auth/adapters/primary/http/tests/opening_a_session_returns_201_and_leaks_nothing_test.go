package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestOpeningASessionReturns201AndLeaksNothing éprouve le chemin heureux ET la
// discrétion de la réponse.
//
// # Ce que le corps BRUT ajoute
//
// L'inspection champ par champ passerait à côté d'un secret transporté sous un
// nom anodin. La recherche dans le texte, elle, ne dépend d'aucune supposition
// sur la forme de la fuite.
func TestOpeningASessionReturns201AndLeaksNothing(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	resp := openSession(t, server, subject, secret)

	if resp.status != http.StatusCreated {
		t.Fatalf("attendu 201, obtenu %d — corps %s", resp.status, resp.raw)
	}

	token := tokenOf(t, resp)
	if len(token) < 43 {
		t.Fatalf("le jeton doit porter au moins 256 bits d'aléa, obtenu %d caractères", len(token))
	}
	if _, ok := resp.body["identity_id"].(string); !ok {
		t.Fatalf("la réponse doit porter l'identifiant : %s", resp.raw)
	}
	if _, ok := resp.body["expires_at"].(string); !ok {
		t.Fatalf("la réponse doit borner la session dans le temps : %s", resp.raw)
	}

	if strings.Contains(resp.raw, secret) {
		t.Fatalf("le secret EN CLAIR fuite dans la réponse : %s", resp.raw)
	}
	if strings.Contains(resp.raw, "condensé") {
		t.Fatalf("le condensé fuite dans la réponse : %s", resp.raw)
	}

	// Aucune permission, aucun rôle : le jeton authentifie, il n'autorise pas.
	// Les y publier inviterait tout client à les mettre en cache, et la
	// révocation cesserait d'être immédiate sans qu'une ligne de ce dépôt change.
	for _, forbidden := range []string{"permission", "permissions", "roles", "scope", "scopes"} {
		if _, present := resp.body[forbidden]; present {
			t.Fatalf("la réponse de connexion porte %q : %s", forbidden, resp.raw)
		}
	}
}

// TestTwoSessionsGetTwoDistinctTokens garde l'aléa du jeton.
//
// Deux connexions du même compte doivent produire deux jetons différents. Un
// jeton dérivé du sujet — ou pire, constant — serait devinable, et une
// authentification devinable n'en est pas une.
func TestTwoSessionsGetTwoDistinctTokens(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	first := tokenOf(t, openSession(t, server, subject, secret))
	second := tokenOf(t, openSession(t, server, subject, secret))

	if first == second {
		t.Fatal("deux connexions ont produit le même jeton")
	}
	if strings.Contains(first, subject) {
		t.Fatalf("le jeton dérive du sujet : %q", first)
	}
}
