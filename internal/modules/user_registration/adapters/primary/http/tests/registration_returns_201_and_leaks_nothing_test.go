package tests

import (
	"net/http"
	"strings"
	"testing"
)

// TestRegistrationReturns201AndLeaksNothing : le chemin nominal, et surtout ce
// que la réponse NE contient PAS.
//
// # Les deux moitiés de ce test
//
// La première est banale : 201, et les champs attendus.
//
// La seconde ne l'est pas. Le corps est sérialisé depuis le type du LANGAGE
// PUBLIÉ, pas depuis `domain.User`. Le jour où quelqu'un « simplifie » en
// renvoyant directement l'entité du domaine, le condensé du mot de passe part
// dans la réponse HTTP — et rien ne le signale, parce que la réponse reste un
// JSON valide avec les bons champs en plus.
//
// Ce test lit le corps BRUT et refuse d'y trouver le condensé. C'est le seul
// moyen d'attraper une fuite par ajout de champ.
func TestRegistrationReturns201AndLeaksNothing(t *testing.T) {
	t.Parallel()

	server := newServer(t)
	resp := post(t, server, registerBody(t, "Alice@Example.COM ", validPassword))

	if resp.status != http.StatusCreated {
		t.Fatalf("statut = %d, attendu 201", resp.status)
	}
	body, raw := resp.body, resp.raw

	// L'adresse est normalisée par le DOMAINE — casse et espaces compris. La
	// surface ne fait aucune normalisation : si elle en faisait une, les deux
	// divergeraient un jour.
	if got := body["email"]; got != "alice@example.com" {
		t.Errorf("email = %v, attendu l'adresse normalisée alice@example.com", got)
	}
	// Un compte naît PENDING. Naître actif serait un fail-open : l'adresse n'est
	// pas encore prouvée.
	if got := body["status"]; got != "pending" {
		t.Errorf("status = %v, attendu pending — un compte ne naît jamais actif", got)
	}
	if got, ok := body["user_id"].(string); !ok || got == "" {
		t.Errorf("user_id = %v, attendu un identifiant non vide", body["user_id"])
	}

	// Recherche dans le TEXTE de la réponse, pas dans les noms de champs : un
	// condensé transporté par un champ au nom anodin échapperait à toute
	// inspection champ par champ.
	forbidden := map[string]string{
		"le condensé du mot de passe": "hashed:",
		"le mot de passe en clair":    validPassword,
	}
	for what, needle := range forbidden {
		if strings.Contains(raw, needle) {
			t.Errorf("%s figure dans la réponse HTTP: %s", what, raw)
		}
	}
	if _, present := body["password_hash"]; present {
		t.Error("la réponse expose password_hash — le corps doit venir du langage publié, pas du domaine")
	}
}
