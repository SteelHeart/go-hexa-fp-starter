package tests

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestInvalidCursorIsRefused : un curseur illisible ou falsifié est REFUSÉ.
//
// Le curseur vient du client, donc d'un inconnu. Se rabattre silencieusement sur
// « première page » sur une entrée corrompue produirait une boucle infinie côté
// client : il redemanderait éternellement la même première page en croyant avancer.
//
// Toutes les causes rendent la même erreur sentinelle, pour que l'adaptateur HTTP
// puisse répondre 400 sans connaître le détail du format.
func TestInvalidCursorIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"vide":                     "",
		"base64 invalide":          "!!!pas du base64!!!",
		"sans séparateur":          base64.RawURLEncoding.EncodeToString([]byte("1234567890")),
		"identifiant manquant":     base64.RawURLEncoding.EncodeToString([]byte("1234567890|")),
		"horodatage non numérique": base64.RawURLEncoding.EncodeToString([]byte("hier|user-42")),
		"tout vide":                base64.RawURLEncoding.EncodeToString([]byte("|")),
	}

	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := pagination.DecodeCursor(encoded); !errors.Is(err, pagination.ErrInvalidCursor) {
				t.Errorf("DecodeCursor = %v, attendu ErrInvalidCursor", err)
			}
		})
	}
}
