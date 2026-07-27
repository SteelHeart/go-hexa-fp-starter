package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRateLimiterIsolatesEachClient : un client qui sature n'affecte pas les autres.
//
// # Le défaut que ce test attrape
//
// Un limiteur qui compte GLOBALEMENT au lieu de compter PAR CLIENT transforme la
// protection en arme : un seul attaquant suffit alors à mettre tout le monde en
// 429. Le service reste debout, répond, ne journalise aucune erreur — et personne
// ne peut s'en servir.
//
// Ce défaut ne se voit jamais en développement, où il n'y a qu'un client.
func TestRateLimiterIsolatesEachClient(t *testing.T) {
	t.Parallel()

	const burst = 2
	limiter := middleware.NewRateLimiter(0.001, burst, time.Minute).Middleware()

	// Le premier client épuise son quota.
	for i := range burst {
		if code := requestFrom(t, limiter, "10.0.0.1:1234"); code != http.StatusOK {
			t.Fatalf("requête %d du premier client = %d, attendu 200", i+1, code)
		}
	}
	if code := requestFrom(t, limiter, "10.0.0.1:1234"); code != http.StatusTooManyRequests {
		t.Errorf("au-delà du quota = %d, attendu 429", code)
	}

	// Un AUTRE client doit conserver le sien.
	if code := requestFrom(t, limiter, "10.0.0.2:5678"); code != http.StatusOK {
		t.Errorf("second client = %d, attendu 200 — le quota est PAR CLIENT", code)
	}
}

// requestFrom envoie une requête depuis une adresse donnée.
func requestFrom(t *testing.T, mw func(http.Handler) http.Handler, remoteAddr string) int {
	t.Helper()

	req := get(t)
	req.RemoteAddr = remoteAddr
	return call(mw, req, okHandler(nil)).Code
}
