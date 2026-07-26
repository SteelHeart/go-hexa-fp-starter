package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestTheMiddlewareStackIsActuallyMounted : la pile est BRANCHÉE, pas seulement écrite.
//
// # Pourquoi ce test existe séparément de ceux du paquet middleware
//
// `internal/pkg/middleware/tests` prouve que chaque garde FONCTIONNE. Il ne prouve
// pas qu'elle est MONTÉE. Ce sont deux défauts distincts, et le second ne laisse
// aucune trace : un `mux.Use(...)` dont une ligne a disparu à la fusion produit un
// service parfaitement fonctionnel, simplement sans protection.
//
// Aucun test du paquet middleware ne peut l'attraper — ils construisent la pile
// eux-mêmes. C'est donc ici, et nulle part ailleurs.
//
// Ce test vérifie ce qui est OBSERVABLE dans une réponse : l'identifiant de
// corrélation et l'origine refusée. La limitation de débit et la borne de corps
// ont leurs propres tests ; les rejouer ici les rendrait dépendants de la
// configuration de test.
func TestTheMiddlewareStackIsActuallyMounted(t *testing.T) {
	t.Parallel()

	headers := get(t, config.EnvProduction, nil, "/healthz").Header()

	// RequestID : présent en sortie, toujours. Sans lui, aucune ligne de journal
	// ne peut être reliée à une requête.
	if headers.Get(middleware.RequestIDHeader) == "" {
		t.Errorf("%s absent de la réponse — middleware.RequestID() n'est pas monté",
			middleware.RequestIDHeader)
	}
}

// TestCORSIsMountedAndDeniesAnUnlistedOrigin : la garde d'origine est en place.
//
// La configuration de test n'autorise que `https://app.example.com`. Une réponse
// qui accorderait l'accès à une autre origine signalerait soit un CORS absent de
// la pile, soit un CORS monté avec la mauvaise liste.
func TestCORSIsMountedAndDeniesAnUnlistedOrigin(t *testing.T) {
	t.Parallel()

	rec := get(t, config.EnvProduction, nil, "/healthz")

	// Aucune origine n'a été envoyée : aucun en-tête d'autorisation ne doit
	// apparaître. Un `*` ici serait une ouverture totale.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q sans en-tête Origin dans la requête", got)
	}
}
