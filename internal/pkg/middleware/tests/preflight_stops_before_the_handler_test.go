package tests

import (
	"net/http"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestPreflightStopsBeforeTheHandler : un OPTIONS s'arrête au middleware.
//
// Laisser la pré-vérification atteindre le gestionnaire lui ferait exécuter la
// logique métier d'une requête qui n'est PAS la vraie requête — le navigateur
// n'a encore rien envoyé d'utile. Sur une route d'écriture, cela reviendrait à
// exécuter l'opération deux fois, dont une sans corps.
func TestPreflightStopsBeforeTheHandler(t *testing.T) {
	t.Parallel()

	const origin = "https://app.example.com"

	reached := false
	rec := call(middleware.CORS([]string{origin}), preflight(t, origin), okHandler(&reached))

	if reached {
		t.Error("le gestionnaire a été atteint par une pré-vérification OPTIONS")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("statut = %d, attendu 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("la pré-vérification doit annoncer les méthodes autorisées")
	}
}
