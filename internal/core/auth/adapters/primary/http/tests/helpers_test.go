// Package tests éprouve la SURFACE HTTP du module d'authentification.
//
// # En processus, sans aucune infrastructure
//
// Le serveur est monté en mémoire par `httptest` : ni port ouvert, ni binaire
// lancé, ni service externe. Ces tests tournent donc dans `go test ./...`
// ordinaire, sur n'importe quelle machine, en quelques millisecondes — et ils
// cassent à la seconde où quelqu'un change un code de statut.
//
// Ce qu'ils vérifient : la TRADUCTION, et les fuites. Le domaine et les cas
// d'usage ont leurs propres tests ; ici on vérifie qu'un refus devient le bon
// statut, qu'un secret ne ressort jamais, et que deux refus différents restent
// indiscernables une fois traduits.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	authhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/adapters/primary/http"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

const (
	// subject désigne le compte utilisé par les tests.
	subject = "alice@example.com"

	// secret satisfait les bornes du module.
	secret = "correct cheval batterie agrafe"

	// sessionsPath et identityPath doublent volontairement les constantes du
	// contrat : un test qui les importerait resterait vert si quelqu'un
	// renommait une route, et un changement de chemin est cassant pour tout
	// client déjà déployé.
	sessionsPath   = "/v1/auth/sessions"
	currentPath    = "/v1/auth/sessions/current"
	identityPath   = "/v1/auth/identity"
	identitiesPath = "/v1/auth/identities"
	rolePath       = "/v1/auth/roles/admin"

	// jetonFactice a la BONNE forme et n'a jamais été émis : 43 caractères,
	// exactement ce que le domaine exige. C'est ce qui permet d'éprouver le
	// refus d'un jeton bien formé mais inconnu.
	jetonFactice = "0000000000000000000000000000000000000000000"
)

// hashSecret est un hachage FACTICE, instantané.
//
// Argon2id est délibérément lent. Le payer ici ferait durer chaque test des
// dizaines de millisecondes sans rien éprouver de plus : ce qui est teste, c'est
// la traduction HTTP, pas la solidité du condensé.
func hashSecret(plain string) (string, error) { return "condensé:" + plain, nil }

func verifySecret(plain, encoded string) (bool, error) { return encoded == "condensé:"+plain, nil }

// newServer monte la surface sur un serveur en mémoire, avec un compte inscrit.
//
// Le module tourne sur son pilote par défaut : chaque test a donc son propre
// magasin, vierge, et peut s'exécuter en parallèle sans interférer.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	mod := newModule(t, config.Module{Enabled: true, Driver: "memory"})
	if _, err := mod.Register(context.Background(), subject, secret); err != nil {
		t.Fatalf("inscription: %v", err)
	}
	return serve(t, mod)
}

// newModule construit le module ou fait échouer le test.
func newModule(t *testing.T, cfg config.Module) auth.Module {
	t.Helper()

	mod, err := auth.New(cfg, auth.Deps{
		HashSecret:   hashSecret,
		VerifySecret: verifySecret,
		Now:          func() time.Time { return time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("montage du module: %v", err)
	}
	return mod
}

// serve monte un module sur un serveur en mémoire.
func serve(t *testing.T, mod auth.Module) *httptest.Server {
	t.Helper()

	router := httpserver.NewRouter(testConfig(), discardLogger(), nil)
	authhttp.Mount(router.API, mod)

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
		App:    config.App{Env: config.EnvTest, Name: "surface-auth-test", Version: "test"},
		HTTP:   config.HTTP{MaxBodyBytes: 1 << 20},
		Limits: config.Limits{RPS: 10_000, Burst: 10_000},
	}
}

// discardLogger tait les journaux du routeur pendant les tests.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// response est une réponse HTTP entièrement consommée.
//
// Le corps BRUT est conservé à côté du décodé, et il compte autant : une fuite se
// cherche dans le TEXTE, pas dans une carte. Un secret transporté par un champ au
// nom anodin échapperait à toute inspection champ par champ.
type response struct {
	status int
	body   map[string]any
	raw    string
}

// openSession demande une session et rend la réponse, corps lu et fermé.
func openSession(t *testing.T, server *httptest.Server, subj, sec string) response {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"subject": subj, "secret": sec})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}

	// bodyclose ne suit pas la fermeture faite par `consume`, qui la fait
	// pourtant en `defer` sur tous les chemins.
	//
	//nolint:noctx,bodyclose // httptest local ; le corps est lu et fermé par consume
	resp, err := server.Client().Post(
		server.URL+sessionsPath, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST %s: %v", sessionsPath, err)
	}
	return consume(t, resp)
}

// withBearer envoie une requête portant un en-tête `Authorization`.
//
// L'en-tête est passé BRUT et non construit à partir d'un jeton : c'est ce qui
// permet d'éprouver un schéma absent, un schéma inconnu, ou une casse
// inattendue — les cas où la surface décide seule, sans le module.
func withBearer(t *testing.T, server *httptest.Server, method, path, header string) response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, server.URL+path, http.NoBody)
	if err != nil {
		t.Fatalf("construction de la requête: %v", err)
	}
	if header != "" {
		req.Header.Set("Authorization", header)
	}

	//nolint:bodyclose // le corps est lu et fermé par consume
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return consume(t, resp)
}

// consume lit intégralement la réponse et la ferme.
//
// Un corps vide est ACCEPTÉ sans erreur : 204 n'en a pas, et faire échouer le
// test sur son absence obligerait chaque appelant à distinguer les cas.
func consume(t *testing.T, resp *http.Response) response {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("lecture du corps: %v", err)
	}

	var payload map[string]any
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("corps de réponse illisible (%q): %v", raw, err)
		}
	}
	return response{status: resp.StatusCode, body: payload, raw: string(raw)}
}

// serveurAmorce monte la surface AVEC un compte d'amorçage administrateur.
//
// C'est le seul moyen d'éprouver une route protégée : sans rôle, tout rend 403,
// et un test qui ne verrait que des 403 ne distinguerait pas un garde correct
// d'un garde qui refuse tout.
func serveurAmorce(t *testing.T) (server *httptest.Server, jetonAdmin string) {
	t.Helper()

	mod := newModule(t, config.Module{Enabled: true, Driver: "memory"})
	report, err := auth.Bootstrap(context.Background(), mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("amorçage: %v", err)
	}
	if !report.Created {
		t.Fatal("l'amorçage devait créer le compte administrateur")
	}

	server = serve(t, mod)
	return server, tokenOf(t, openSession(t, server, report.Subject, report.Secret))
}

// requete décrit un appel HTTP complet.
//
// Une structure plutôt que six paramètres : la règle de forme en autorise cinq,
// et surtout `entete` et `corps` sont deux chaînes voisines — les inverser
// compile, et produit une requête sans corps portant un JSON en guise
// d'autorisation. Nommer les champs rend l'inversion visible.
type requete struct {
	methode string
	chemin  string
	entete  string
	corps   string
}

// porteur construit l'en-tête d'autorisation d'un jeton.
func porteur(token string) string { return "Bearer " + token }

// envoyer exécute la requête et rend la réponse, corps lu et fermé.
func envoyer(t *testing.T, server *httptest.Server, r requete) response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(), r.methode, server.URL+r.chemin, strings.NewReader(r.corps))
	if err != nil {
		t.Fatalf("construction de la requête: %v", err)
	}
	if r.corps != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if r.entete != "" {
		req.Header.Set("Authorization", r.entete)
	}

	//nolint:bodyclose // le corps est lu et fermé par consume
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", r.methode, r.chemin, err)
	}
	return consume(t, resp)
}

// tokenOf extrait le jeton d'une réponse de connexion.
func tokenOf(t *testing.T, resp response) string {
	t.Helper()

	token, ok := resp.body["token"].(string)
	if !ok || token == "" {
		t.Fatalf("la réponse ne porte pas de jeton exploitable : %s", resp.raw)
	}
	return token
}
