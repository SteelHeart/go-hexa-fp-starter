package tests

import (
	"net/http"
	"testing"
)

// TestATakenAddressAnswers409Not422 : une adresse déjà prise rend 409, pas 422.
//
// # La distinction que ce test protège
//
// 422 dit « ta requête est mal formée ». 409 dit « ta requête est parfaitement
// formée, c'est l'état du serveur qui s'y oppose ». Ce ne sont pas deux façons
// de dire la même chose :
//
//   - Un formulaire d'inscription doit afficher « cette adresse est déjà
//     utilisée, connectez-vous » sur un 409, et « adresse invalide » sur un 422.
//     Les confondre produit le message le plus frustrant qui soit : dire à
//     quelqu'un que son adresse est invalide alors qu'elle est simplement à lui.
//   - Un client automatisé peut traiter le 409 comme un succès idempotent ; il
//     ne peut jamais faire ça d'un 422.
//
// Le test inscrit d'abord, puis rejoue avec une casse DIFFÉRENTE : c'est le
// domaine qui normalise, donc `ALICE@EXAMPLE.COM` doit entrer en collision avec
// `alice@example.com`. Sans normalisation, on créerait deux comptes pour une
// même personne — et le doublon ne se verrait qu'au support.
func TestATakenAddressAnswers409Not422(t *testing.T) {
	t.Parallel()

	server := newServer(t)

	first := post(t, server, registerBody(t, "alice@example.com", validPassword))
	if first.status != http.StatusCreated {
		t.Fatalf("première inscription: statut = %d, attendu 201", first.status)
	}

	resp := post(t, server, registerBody(t, "  ALICE@Example.com ", validPassword))
	if resp.status != http.StatusConflict {
		t.Fatalf("statut = %d, attendu 409 — l'adresse normalisée est déjà prise", resp.status)
	}
	if got := resp.body["detail"]; got != "cette adresse de courriel est déjà enregistrée" {
		t.Errorf("detail = %v, attendu le message du domaine", got)
	}
	assertFieldLocated(t, resp.body, "body.email")
}
