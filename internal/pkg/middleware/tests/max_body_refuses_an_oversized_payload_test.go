package tests

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestMaxBodyRefusesAnOversizedPayload : la lecture s'arrête à la borne.
//
// Sans borne, un client fait grossir la mémoire du serveur à volonté : il suffit
// d'ouvrir une requête et d'envoyer indéfiniment. Ce n'est pas une attaque
// sophistiquée, c'est `curl` avec un fichier assez gros — et le serveur meurt
// d'épuisement mémoire, pas d'une erreur qu'on pourrait diagnostiquer.
//
// Le test lit RÉELLEMENT le corps : la borne ne s'applique qu'à la lecture, donc
// un gestionnaire qui ne lit pas ne prouve rien.
func TestMaxBodyRefusesAnOversizedPayload(t *testing.T) {
	t.Parallel()

	const limit = 64

	var readErr error
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	})

	call(middleware.MaxBody(limit), post(t, strings.Repeat("a", limit*4)), reader)

	if readErr == nil {
		t.Error("un corps au-delà de la borne doit faire échouer la lecture")
	}
}

// TestMaxBodyLetsAnAcceptablePayloadThrough : sous la borne, rien ne change.
//
// Une borne qui refuserait aussi le trafic légitime serait retirée dès la
// première plainte — et la protection avec elle.
func TestMaxBodyLetsAnAcceptablePayloadThrough(t *testing.T) {
	t.Parallel()

	const limit = 64
	payload := strings.Repeat("a", limit/2)

	var got string
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("lecture sous la borne en échec: %v", err)
		}
		got = string(raw)
	})

	call(middleware.MaxBody(limit), post(t, payload), reader)

	if got != payload {
		t.Errorf("corps lu = %d octets, attendu %d", len(got), len(payload))
	}
}
