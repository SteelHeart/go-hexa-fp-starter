package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestLastPageHasNoNextCursor : la dernière page ne propose pas de suite.
//
// Rendre un curseur alors qu'il n'y a plus rien ferait boucler un client qui suit
// les curseurs jusqu'à épuisement : il redemanderait indéfiniment une page vide qui
// lui rendrait à nouveau un curseur. `HasMore` et `NextCursor` doivent donc dire la
// même chose, toujours.
func TestLastPageHasNoNextCursor(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	cases := map[string]int{"page incomplète": 2, "page exactement pleine": 3}

	for name, count := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page := pagination.NewPage(lignes(count), req, curseurDe)
			if page.HasMore {
				t.Error("HasMore doit être faux : aucune ligne supplémentaire n'a été lue")
			}
			if page.NextCursor != "" {
				t.Errorf("NextCursor = %q, attendu vide sur la dernière page", page.NextCursor)
			}
			if len(page.Items) != count {
				t.Errorf("éléments = %d, attendu %d", len(page.Items), count)
			}
		})
	}
}
