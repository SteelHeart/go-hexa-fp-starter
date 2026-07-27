package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRateLimiterIgnoresAForgeableHeader : `X-Forwarded-For` ne change pas
// l'identité du client.
//
// # Pourquoi c'est une décision de sécurité, pas un oubli
//
// Prendre `X-Forwarded-For` comme identité rendrait la limitation TRIVIALEMENT
// contournable : il suffit d'envoyer une valeur différente à chaque requête pour
// obtenir un quota neuf à chaque fois. La garde aurait l'air de fonctionner —
// elle refuserait bien au-delà du quota si on ne trichait pas — tout en ne
// protégeant de rien contre quiconque essaie.
//
// L'en-tête n'est exploitable que réécrit par un mandataire de confiance. Tant
// que ce mandataire n'est pas décrit dans la configuration, il n'existe pas, et
// seule `RemoteAddr` fait foi.
func TestRateLimiterIgnoresAForgeableHeader(t *testing.T) {
	t.Parallel()

	const burst = 2
	limiter := middleware.NewRateLimiter(0.001, burst, time.Minute).Middleware()

	for range burst {
		_ = requestFrom(t, limiter, "10.0.0.9:1111")
	}

	// Même adresse réelle, en-tête falsifié à chaque appel : le quota doit rester
	// épuisé.
	for attempt := range 3 {
		req := get(t)
		req.RemoteAddr = "10.0.0.9:1111"
		req.Header.Set("X-Forwarded-For", "203.0.113."+string(rune('1'+attempt)))

		if code := call(limiter, req, okHandler(nil)).Code; code != http.StatusTooManyRequests {
			t.Fatalf("tentative %d = %d, attendu 429 — X-Forwarded-For ne doit pas offrir un quota neuf",
				attempt+1, code)
		}
	}
}
