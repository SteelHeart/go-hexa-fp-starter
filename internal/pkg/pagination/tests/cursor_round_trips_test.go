package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestCursorRoundTrips : encoder puis décoder rend le curseur d'origine.
//
// C'est la propriété dont dépend toute la pagination : un curseur fait l'aller-
// retour par le réseau, et s'il ne revient pas identique, l'appelant reprend à un
// endroit qui n'est pas celui qu'il avait quitté — donc saute ou répète des lignes.
//
// La précision est à la MICROSECONDE, délibérément : c'est celle de `timestamptz`
// côté PostgreSQL. Conserver des nanosecondes en mémoire produirait un curseur que
// la base ne sait pas retrouver.
func TestCursorRoundTrips(t *testing.T) {
	t.Parallel()

	cases := map[string]pagination.Cursor{
		"instant ordinaire": {CreatedAt: base(), ID: "user-42"},
		"epoch":             {CreatedAt: time.UnixMicro(0).UTC(), ID: "x"},
		"identifiant long":  {CreatedAt: base(), ID: "01J8ZQ9V3K4M5N6P7R8S9T0V1W"},
		"avant epoch":       {CreatedAt: time.UnixMicro(-1_000_000).UTC(), ID: "ancien"},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := pagination.DecodeCursor(want.Encode())
			if err != nil {
				t.Fatalf("décodage: %v", err)
			}
			if !got.CreatedAt.Equal(want.CreatedAt) {
				t.Errorf("horodatage = %v, attendu %v", got.CreatedAt, want.CreatedAt)
			}
			if got.ID != want.ID {
				t.Errorf("identifiant = %q, attendu %q", got.ID, want.ID)
			}
		})
	}
}
