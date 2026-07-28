package tests

import (
	"net/http"
	"testing"
)

// TestEveryRefusalAnswersTheSame401 ferme l'oracle À LA SURFACE.
//
// # Pourquoi le rejouer ici alors que le cas d'usage le garantit déjà
//
// Parce que la surface a sa propre occasion de le rompre. Le module rend une
// seule sentinelle, mais c'est le traducteur qui choisit le statut ET le message
// — et il suffit d'un `err.Error()` renvoyé tel quel, ou d'un 404 « compte
// inconnu » ajouté par bonne intention, pour rouvrir ce que le domaine avait
// fermé.
//
// Le test compare les réponses ENTRE ELLES, pas à une valeur attendue : c'est
// l'indiscernabilité qui est la propriété.
func TestEveryRefusalAnswersTheSame401(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	cases := map[string][2]string{
		"sujet inconnu":   {"personne@example.com", secret},
		"secret faux":     {subject, "un-autre-secret-bien-assez-long"},
		"sujet vide":      {"", secret},
		"secret vide":     {subject, ""},
		"sujet mal formé": {"   ", secret},
	}

	bodies := make(map[string]bool)
	for name, tc := range cases {
		resp := openSession(t, server, tc[0], tc[1])
		if resp.status != http.StatusUnauthorized {
			t.Fatalf("%s : attendu 401, obtenu %d — corps %s", name, resp.status, resp.raw)
		}
		bodies[resp.raw] = true
	}

	if len(bodies) != 1 {
		t.Fatalf("les refus doivent être indiscernables ; %d corps distincts : %v", len(bodies), bodies)
	}
}

// TestAMalformedBearerNeverSaysWhy applique la même règle au jeton.
//
// En-tête absent, schéma inconnu, jeton trop court, jeton inventé : un seul
// statut et un seul corps. Distinguer « mal formé » de « inconnu » dirait à un
// attaquant que sa chaîne a la bonne FORME — donc qu'il approche.
func TestAMalformedBearerNeverSaysWhy(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	headers := map[string]string{
		"absent":            "",
		"schéma manquant":   "0123456789012345678901234567890123456789012",
		"schéma inconnu":    "Basic 0123456789012345678901234567890123456789012",
		"jeton trop court":  "Bearer trop-court",
		"jeton vide":        "Bearer ",
		"jeton inventé":     "Bearer 9999999999999999999999999999999999999999999",
		"schéma sans jeton": "Bearer",
	}

	bodies := make(map[string]bool)
	for name, header := range headers {
		resp := withBearer(t, server, http.MethodGet, identityPath, header)
		if resp.status != http.StatusUnauthorized {
			t.Fatalf("%s : attendu 401, obtenu %d — corps %s", name, resp.status, resp.raw)
		}
		bodies[resp.raw] = true
	}

	if len(bodies) != 1 {
		t.Fatalf("les refus de jeton doivent être indiscernables ; %d corps : %v", len(bodies), bodies)
	}
}

// TestTheBearerSchemeIsCaseInsensitive suit la RFC 7235.
//
// Le schéma d'authentification y est déclaré insensible à la casse. Refuser
// `bearer ` ferait échouer des clients parfaitement corrects, avec un 401 qui
// accuserait le jeton — donc en envoyant chercher la panne au mauvais endroit.
func TestTheBearerSchemeIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	token := tokenOf(t, openSession(t, server, subject, secret))

	for _, scheme := range []string{"Bearer ", "bearer ", "BEARER ", "BeArEr "} {
		resp := withBearer(t, server, http.MethodGet, identityPath, scheme+token)
		if resp.status != http.StatusOK {
			t.Errorf("schéma %q : attendu 200, obtenu %d — corps %s", scheme, resp.status, resp.raw)
		}
	}
}
