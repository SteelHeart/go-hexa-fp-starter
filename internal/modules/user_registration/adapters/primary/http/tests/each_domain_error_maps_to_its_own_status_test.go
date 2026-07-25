package tests

import (
	"net/http"
	"testing"
)

// TestEachDomainErrorMapsToItsOwnStatus : chaque refus du domaine produit SON
// statut, et le message rendu vient du domaine.
//
// # Pourquoi la distinction des statuts n'est pas cosmétique
//
// Un client automatisé décide de réessayer ou non à partir du statut :
//
//   - **422** — la requête est mal formée. Réessayer à l'identique échouera
//     toujours. Le client doit corriger, ou renoncer.
//   - **409** — la requête est BIEN formée, c'est l'état du serveur qui s'y
//     oppose. Le client sait que ce n'est pas sa faute.
//   - **503** — transitoire. Réessayer plus tard a du sens.
//
// Tout rendre en 400, ou pire en 500, force chaque client à analyser un message
// en français pour décider. Et un 500 sur une faute de saisie réveille quelqu'un
// la nuit pour un utilisateur qui a mal tapé son adresse.
//
// Le test vérifie aussi que le message vient du DOMAINE. Une contrainte de
// validation posée dans le schéma HTTP court-circuiterait le domaine et rendrait
// le message du framework — en anglais, au milieu d'une API francophone. Ce
// défaut a réellement existé ici avant d'être retiré.
func TestEachDomainErrorMapsToItsOwnStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		email, password string
		wantStatus      int
		wantDetail      string
		wantField       string
	}{
		"adresse invalide": {
			email: "pas-une-adresse", password: validPassword,
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "l'adresse de courriel n'est pas valide",
			wantField:  "body.email",
		},
		"mot de passe trop court": {
			email: "bob@example.com", password: "court",
			wantStatus: http.StatusUnprocessableEntity,
			wantDetail: "le mot de passe doit faire au moins 12 caractères",
			wantField:  "body.password",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := newServer(t)
			resp := post(t, server, registerBody(t, tc.email, tc.password))

			if resp.status != tc.wantStatus {
				t.Fatalf("statut = %d, attendu %d", resp.status, tc.wantStatus)
			}
			if got := resp.body["detail"]; got != tc.wantDetail {
				t.Errorf("detail = %v, attendu le message du domaine %q", got, tc.wantDetail)
			}
			assertFieldLocated(t, resp.body, tc.wantField)
		})
	}
}

// assertFieldLocated vérifie que la réponse nomme le champ fautif.
//
// Sans lui, un formulaire ne sait pas quoi surligner et affiche l'erreur en haut
// de page — ce qui, sur un formulaire à deux champs, oblige l'utilisateur à
// deviner lequel reprendre.
func assertFieldLocated(t *testing.T, body map[string]any, want string) {
	t.Helper()

	errs, ok := body["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Fatalf("la réponse ne porte aucun détail d'erreur: %v", body)
	}
	first, ok := errs[0].(map[string]any)
	if !ok {
		t.Fatalf("détail d'erreur illisible: %v", errs[0])
	}
	if got := first["location"]; got != want {
		t.Errorf("location = %v, attendu %q — le champ fautif doit être nommé", got, want)
	}
}
