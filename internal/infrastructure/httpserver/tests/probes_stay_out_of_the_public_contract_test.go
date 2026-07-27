package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// TestProbesStayOutOfThePublicContract : les sondes ne sont pas dans l'OpenAPI.
//
// # Pourquoi ça compte
//
// Le contrat OpenAPI est ce que consomment les clients : générateurs de SDK,
// portails, tests de contrat. Les sondes n'en font pas partie — elles sont un
// détail d'exploitation, susceptible de changer sans préavis.
//
// Si elles y entraient, un générateur produirait des méthodes `healthz()` dans
// chaque SDK publié, et les retirer deviendrait une RUPTURE de contrat. Une
// décision d'exploitation se serait transformée en engagement public, sans que
// personne l'ait voulu.
//
// C'est pourquoi elles sont montées sur le `*chi.Mux` directement, et non via
// `huma.Register`. Ce test verrouille cette distinction, qui ne se voit pas en
// lisant `NewRouter`.
func TestProbesStayOutOfThePublicContract(t *testing.T) {
	t.Parallel()

	router := router(config.EnvProduction, map[string]httpserver.Probe{"database": healthyProbe()})

	// La description OpenAPI est construite par huma à partir des routes
	// ENREGISTRÉES auprès de lui. Les sondes n'en étant pas, elle doit être vide
	// de tout chemin.
	paths := router.API.OpenAPI().Paths
	for path := range paths {
		if strings.Contains(path, "healthz") || strings.Contains(path, "readyz") {
			t.Errorf("la sonde %q figure dans le contrat OpenAPI — un SDK généré l'exposerait", path)
		}
	}
}

// TestProbesRespondEvenThoughTheyAreUndocumented : hors contrat ≠ absent.
//
// Le pendant du test précédent : une sonde retirée du contrat pourrait aussi
// avoir été retirée du routeur. Les deux tests ensemble disent la seule chose
// vraie — montée, mais non publiée.
func TestProbesRespondEvenThoughTheyAreUndocumented(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/healthz", "/readyz"} {
		if code := get(t, config.EnvProduction, nil, path).Code; code == 404 {
			t.Errorf("%s rend 404 — la sonde n'est pas montée", path)
		}
	}
}
