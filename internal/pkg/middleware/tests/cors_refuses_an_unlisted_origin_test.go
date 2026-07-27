package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSRefusesAnUnlistedOrigin : une origine non listée n'obtient AUCUN
// en-tête d'autorisation.
//
// # Pourquoi c'est le test le plus important du paquet
//
// Un CORS trop permissif ne casse rien. La page fonctionne, les journaux sont
// propres, aucune alerte ne part. Il se découvre le jour où un site tiers lit
// les réponses authentifiées d'un utilisateur connecté — et à ce moment-là, la
// fuite a déjà eu lieu.
//
// C'est exactement le profil du défaut qu'un test doit garder : invisible tant
// qu'il n'est pas exploité. Les cas ci-dessous sont ceux qu'une comparaison
// approximative — préfixe, suffixe, « contient » — laisserait passer.
func TestCORSRefusesAnUnlistedOrigin(t *testing.T) {
	t.Parallel()

	allowed := []string{"https://app.example.com"}
	refused := map[string]string{
		"origine inconnue":       "https://evil.example.com",
		"suffixe trompeur":       "https://app.example.com.evil.com",
		"même hôte en clair":     "http://app.example.com",
		"joker":                  "*",
		"préfixe de l'autorisée": "https://app.example.co",
	}

	for name, origin := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set("Origin", origin)
			rec := call(middleware.CORS(allowed), req, okHandler(nil))

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("origine %q autorisée (%q) alors qu'elle n'est pas listée", origin, got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
				t.Errorf("origine %q obtient Allow-Credentials", origin)
			}
		})
	}
}
