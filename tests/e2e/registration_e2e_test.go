//go:build e2e

// Package e2e éprouve le BINAIRE déployé, par-dessus le réseau.
//
// # Ce que ce niveau ajoute, et ce qu'il n'ajoute pas
//
// La surface HTTP a déjà ses tests, en processus, dans
// `internal/modules/user_registration/adapters/primary/http/tests/`. Ils sont
// plus rapides, plus précis, et tournent partout. Répéter ici ce qu'ils
// vérifient n'apporterait rien qu'un doublon à maintenir.
//
// Ce niveau vérifie ce qu'eux ne PEUVENT pas vérifier : que le binaire compilé,
// avec la configuration LIVRÉE, démarre réellement et répond. C'est-à-dire tout
// ce qui se passe entre `go build` et la première requête — chargement de la
// configuration, montage des modules, pile de middlewares, écoute réseau.
//
// C'est exactement la classe de défaut qu'un test en processus laisse passer :
// une configuration invalide, un module qui refuse de se monter, un port qui
// n'écoute pas.
//
// # Exécution
//
//	go test -tags=e2e ./tests/e2e/...
//
// ⚠️ SANS le tag, ce fichier n'est pas compilé : `go test ./tests/e2e/...`
// affiche `ok` en n'exécutant RIEN. Vérifier le code de retour ne suffit pas —
// il faut vérifier que des tests ont réellement tourné.
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// baseURL est l'adresse du serveur lancé par la CI.
func baseURL(t *testing.T) string {
	t.Helper()

	url := os.Getenv("E2E_BASE_URL")
	if url == "" {
		t.Skip("E2E_BASE_URL absent : aucun serveur à interroger")
	}
	return url
}

// TestDeployedBinaryRegistersAUser : le binaire compilé répond à une inscription.
//
// Le parcours complet, par-dessus le réseau : chargement de la configuration
// livrée, montage des modules, pile de middlewares, écoute, traitement.
func TestDeployedBinaryRegistersAUser(t *testing.T) {
	url := baseURL(t)
	client := &http.Client{Timeout: 10 * time.Second}

	// Adresse unique par exécution : la CI peut rejouer le job sans que le
	// magasin, s'il est durable, ne fasse échouer la seconde tentative en 409.
	email := "e2e-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.test"
	body, err := json.Marshal(map[string]string{
		"email":    email,
		"password": "correct cheval batterie agrafe",
	})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}

	resp, err := client.Post(url+"/v1/users", "application/json", bytes.NewReader(body)) //nolint:noctx // client avec délai explicite
	if err != nil {
		t.Fatalf("POST /v1/users: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("statut = %d, attendu 201", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("corps illisible: %v", err)
	}
	if got := payload["status"]; got != "pending" {
		t.Errorf("status = %v, attendu pending", got)
	}
	if id, ok := payload["user_id"].(string); !ok || id == "" {
		t.Errorf("user_id = %v, attendu un identifiant non vide", payload["user_id"])
	}
}

// TestDeployedBinaryServesItsContract : le contrat OpenAPI est servi.
//
// Un contrat que le binaire n'expose pas est un contrat que personne ne peut
// consommer : les SDK clients en découlent. Le chemin NU `/openapi` rend 404 —
// c'est une erreur déjà commise ici, et ce test la fige.
func TestDeployedBinaryServesItsContract(t *testing.T) {
	url := baseURL(t)
	client := &http.Client{Timeout: 10 * time.Second}

	for _, path := range []string{"/openapi.json", "/openapi.yaml", "/docs", "/healthz"} {
		resp, err := client.Get(url + path) //nolint:noctx // client avec délai explicite
		if err != nil {
			t.Errorf("GET %s: %v", path, err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d, attendu 200", path, resp.StatusCode)
		}
	}
}
