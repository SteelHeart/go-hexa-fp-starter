package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestARevokedTokenStopsResolving éprouve la révocation DE BOUT EN BOUT.
//
// Trois requêtes : ouvrir, fermer, réessayer. C'est le parcours qu'un client
// fait réellement, et c'est celui qu'un jeton signé autoportant ne saurait pas
// interrompre sans liste de révocation — donc sans retomber sur le magasin qu'il
// prétendait éviter (ADR 017 § 2 bis).
func TestARevokedTokenStopsResolving(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	if resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token); resp.status != http.StatusOK {
		t.Fatalf("le jeton vient d'être émis : attendu 200, obtenu %d — %s", resp.status, resp.raw)
	}

	resp := withBearer(t, server, http.MethodDelete, currentPath, "Bearer "+token)
	if resp.status != http.StatusNoContent {
		t.Fatalf("fermeture : attendu 204, obtenu %d — %s", resp.status, resp.raw)
	}
	if strings.TrimSpace(resp.raw) != "" {
		t.Fatalf("un 204 ne porte pas de corps, obtenu %q", resp.raw)
	}

	after := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token)
	if after.status != http.StatusUnauthorized {
		t.Fatalf("jeton révoqué : attendu 401, obtenu %d — %s", after.status, after.raw)
	}
}

// TestClosingATwiceClosedSessionStillAnswers204 garde l'idempotence à la surface.
//
// Un client qui se déconnecte deux fois n'a rien fait de mal — un double clic
// suffit. Rendre 401 au second appel ferait afficher une erreur au moment précis
// où l'utilisateur vient de réussir à partir.
func TestClosingATwiceClosedSessionStillAnswers204(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	for attempt := 1; attempt <= 2; attempt++ {
		resp := withBearer(t, server, http.MethodDelete, currentPath, "Bearer "+token)
		if resp.status != http.StatusNoContent {
			t.Fatalf("fermeture n°%d : attendu 204, obtenu %d — %s", attempt, resp.status, resp.raw)
		}
	}
}

// TestResolvingAnIdentityCarriesNoPermission constate ce que la réponse NE porte
// PAS.
//
// Elle dit QUI présente le jeton. Un client qui en déduirait ce qu'il a le droit
// de faire se tromperait au premier retrait de droit — et c'est très exactement
// le jour où l'on ne veut pas se tromper.
func TestResolvingAnIdentityCarriesNoPermission(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	resp := withBearer(t, server, http.MethodGet, identityPath, "Bearer "+token)
	if resp.status != http.StatusOK {
		t.Fatalf("attendu 200, obtenu %d — %s", resp.status, resp.raw)
	}
	if got, _ := resp.body["subject"].(string); got != subject {
		t.Fatalf("attendu le sujet %q, obtenu %q", subject, got)
	}

	for _, forbidden := range []string{"permission", "permissions", "scope", "scopes", "token"} {
		if _, present := resp.body[forbidden]; present {
			t.Fatalf("la réponse porte %q : %s", forbidden, resp.raw)
		}
	}
	if strings.Contains(resp.raw, token) {
		t.Fatalf("le jeton est renvoyé par une route de lecture : %s", resp.raw)
	}
	if strings.Contains(resp.raw, "condensé") {
		t.Fatalf("le condensé fuite : %s", resp.raw)
	}
}
