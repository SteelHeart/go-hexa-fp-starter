package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSAcceptsTheListedOriginAndVaries : l'origine listée passe, et la
// réponse porte `Vary: Origin`.
//
// Deux raisons d'exister :
//
//   - Un refus systématique protégerait tout aussi bien et ne servirait à rien.
//     Le test doit prouver que la garde DISCRIMINE, pas qu'elle bloque.
//   - Sans `Vary: Origin`, un cache partagé — mandataire, CDN — peut servir à une
//     origine la réponse autorisée d'une autre. La garde serait alors contournée
//     par le cache, sans que personne n'ait touché au code.
func TestCORSAcceptsTheListedOriginAndVaries(t *testing.T) {
	t.Parallel()

	const origin = "https://app.example.com"

	req := get(t)
	req.Header.Set("Origin", origin)
	rec := call(middleware.CORS([]string{origin}), req, okHandler(nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("Allow-Origin = %q, attendu %q", got, origin)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, attendu Origin — sinon un cache partagé rejoue l'autorisation", got)
	}
}
