package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSWithAnEmptyListRefusesEverything : liste vide = refus total.
//
// Deny par défaut, et le cas est loin d'être théorique : c'est la configuration
// d'un service qu'on vient de déployer sans avoir encore rempli
// `http.allowed_origins`. Si la liste vide autorisait tout, le moment le plus
// exposé de la vie d'un service — juste après sa mise en ligne, avant que
// quiconque ait relu sa configuration — serait aussi le moins protégé.
func TestCORSWithAnEmptyListRefusesEverything(t *testing.T) {
	t.Parallel()

	for name, allowed := range map[string][]string{
		"liste nil":  nil,
		"liste vide": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set("Origin", "https://app.example.com")
			rec := call(middleware.CORS(allowed), req, okHandler(nil))

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("aucune origine autorisée en configuration, pourtant accordée: %q", got)
			}
		})
	}
}
