package tests

import (
	"net/http"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestADisabledModuleAnswers503Not200 garde le pire scénario par défaut.
//
// # Ce qu'on vérifie vraiment
//
// Qu'un module éteint REFUSE, plutôt que de laisser passer. La surface se monte
// quand même — c'est ce qui permet de répondre clairement au lieu de faire
// échouer le démarrage entier du serveur — et la tentation est alors d'écrire les
// refus « qui comptent » en oubliant les autres.
//
// 503 et non 501 : la capacité EXISTE, elle n'est pas activée sur ce
// déploiement. C'est une décision d'exploitation, pas une fonctionnalité
// manquante, et 503 est ce qui autorise un client à réessayer plus tard.
func TestADisabledModuleAnswers503Not200(t *testing.T) {
	t.Parallel()

	server := serve(t, newModule(t, config.Module{Enabled: false}))
	token := "0123456789012345678901234567890123456789012"

	cases := map[string]response{
		"ouverture": openSession(t, server, subject, secret),
		"résolution": withBearer(
			t, server, http.MethodGet, identityPath, "Bearer "+token),
		"fermeture": withBearer(
			t, server, http.MethodDelete, currentPath, "Bearer "+token),
	}

	for name, resp := range cases {
		if resp.status != http.StatusServiceUnavailable {
			t.Errorf("%s sur module désactivé : attendu 503, obtenu %d — %s", name, resp.status, resp.raw)
		}
	}
}
