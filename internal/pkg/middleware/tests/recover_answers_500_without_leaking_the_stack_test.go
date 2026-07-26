package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRecoverAnswers500WithoutLeakingTheStack : une panique devient un 500
// silencieux sur le détail.
//
// Deux exigences opposées, et il faut les deux :
//
//   - Le processus ne meurt pas. Sans récupération, UNE requête qui panique
//     emporte tout le serveur, donc toutes les requêtes en cours.
//   - Le client n'apprend rien. Une pile d'appels renvoyée expose les chemins de
//     fichiers, les noms de paquets et la structure interne — la carte exacte que
//     cherche quelqu'un qui sonde le service.
func TestRecoverAnswers500WithoutLeakingTheStack(t *testing.T) {
	t.Parallel()

	const secret = "mot_de_passe_dans_la_panique"
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("échec interne avec " + secret)
	})

	rec := call(middleware.Recover(discardLogger()), get(t), panicking)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("statut = %d, attendu 500", rec.Code)
	}

	body := rec.Body.String()
	for _, forbidden := range []string{secret, "goroutine", ".go:", "panic"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("la réponse contient %q — la cause ne doit jamais sortir: %s", forbidden, body)
		}
	}
}
