// Package tests éprouve la SURFACE HTTP du module d'inscription.
//
// # En processus, sans aucune infrastructure
//
// Le serveur est monté en mémoire par `httptest` : ni port ouvert, ni binaire
// lancé, ni service externe. Ces tests tournent donc dans `go test ./...`
// ordinaire, sur n'importe quelle machine, en quelques millisecondes.
//
// C'est délibéré. Un test de surface qui exige d'orchestrer un binaire et une
// base ne tourne qu'en CI ; il est donc écrit une fois, cassé un mois plus tard,
// et personne ne le voit avant la prochaine intégration. Celui-ci casse à la
// seconde où quelqu'un change un code de statut.
//
// Ce qu'ils vérifient : la TRADUCTION. Le domaine a ses propres tests, le
// pipeline aussi. Ici on vérifie uniquement qu'une erreur de domaine devient le
// bon statut HTTP, et qu'aucun champ interne ne fuit dans la réponse.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	userhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/http"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// validPassword satisfait les bornes du domaine.
const validPassword = "correct cheval batterie agrafe"

// newServer monte la surface sur un serveur en mémoire.
//
// Le module tourne sur son pilote par défaut : chaque test a donc son propre
// magasin, vierge, et peut s'exécuter en parallèle sans interférer.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	mod, err := userregistration.New("", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: noopPublish,
		GenerateID:   func() domain.UserID { return "019f9b46-3aec-735a-977d-129192ef130f" },
		Now:          func() time.Time { return time.Date(2026, time.July, 26, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("montage du module: %v", err)
	}

	router := httpserver.NewRouter(testConfig(), discardLogger(), nil)
	userhttp.Mount(router.API, mod)
	userhttp.MountAvailability(router.API, mod)

	server := httptest.NewServer(router.Mux)
	t.Cleanup(server.Close)
	return server
}

// testConfig porte le minimum dont le routeur a besoin.
//
// ⚠️ Les limites de débit sont volontairement HAUTES : à zéro, le limiteur
// refuserait tout et chaque test échouerait en 429 — un défaut dont la cause
// serait invisible dans le message d'erreur.
func testConfig() config.Config {
	return config.Config{
		App: config.App{
			Env:     config.EnvTest,
			Name:    "surface-test",
			Version: "test",
		},
		HTTP:   config.HTTP{MaxBodyBytes: 1 << 20},
		Limits: config.Limits{RPS: 10_000, Burst: 10_000},
	}
}

// response est une réponse HTTP entièrement consommée.
//
// Le corps BRUT est conservé à côté du décodé, et il compte autant : une fuite
// se cherche dans le TEXTE, pas dans une carte. Un condensé transporté par un
// champ au nom anodin échapperait à toute inspection champ par champ.
type response struct {
	status int
	body   map[string]any
	raw    string
}

// post envoie une inscription et rend la réponse, corps lu et fermé.
//
// L'aide ne laisse PAS échapper de `*http.Response`, délibérément : un corps
// laissé ouvert retient une connexion, et confier sa fermeture à chaque test est
// la garantie qu'un test l'oubliera. Ici, la question ne se pose plus.
func post(t *testing.T, server *httptest.Server, body string) response {
	t.Helper()

	// bodyclose ne suit pas la fermeture faite par `consume`, qui la fait
	// pourtant en `defer` sur tous les chemins. Le motif est écrit plutôt que la
	// directive posée sans raison.
	//
	//nolint:noctx,bodyclose // httptest local ; le corps est lu et ferme par consume
	resp, err := server.Client().Post(
		server.URL+"/v1/users", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /v1/users: %v", err)
	}
	return consume(t, resp)
}

// consume lit intégralement la réponse et la ferme.
func consume(t *testing.T, resp *http.Response) response {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("lecture du corps: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("corps de réponse illisible (%q): %v", raw, err)
	}
	return response{status: resp.StatusCode, body: payload, raw: string(raw)}
}

// registerBody construit une charge utile d'inscription.
func registerBody(t *testing.T, email, password string) string {
	t.Helper()

	raw, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("sérialisation de la requête: %v", err)
	}
	return string(raw)
}

// fakeHash évite le coût d'Argon2 : ce n'est pas la cryptographie qu'on teste
// ici, elle a ses propres tests.
func fakeHash(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
	return result.Ok[domain.PasswordHash, domain.Error](
		domain.NewPasswordHash("hashed:" + password.Expose()),
	)
}

// noopPublish accepte tout événement sans rien en faire.
//
// La publication a ses propres tests, dans le module et dans `relay` : ici elle
// ne doit que réussir, pour que le chemin nominal atteigne la réponse HTTP.
func noopPublish(
	_ context.Context, _, _ string, _ any,
) result.Result[domain.Ack, domain.Error] {
	return result.Ok[domain.Ack, domain.Error](domain.Ack{})
}

// discardLogger jette les journaux : un test qui les affiche noie sa propre
// sortie, et ce n'est pas la journalisation qu'on vérifie ici.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
