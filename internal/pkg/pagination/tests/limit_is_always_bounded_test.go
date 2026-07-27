package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestLimitIsAlwaysBounded : une page non bornée est un déni de service offert.
//
// `?limit=1000000` sur une table de plusieurs millions de lignes suffit à saturer
// la mémoire du processus — sans authentification particulière, sans outil, avec
// une seule requête. Le plafond n'est donc pas un réglage de confort.
//
// Zéro et les valeurs négatives retombent sur le défaut plutôt que de refuser :
// une limite absente est le cas nominal d'un premier appel.
func TestLimitIsAlwaysBounded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		demande int
		attendu int
	}{
		"absente":            {demande: 0, attendu: pagination.DefaultLimit},
		"négative":           {demande: -10, attendu: pagination.DefaultLimit},
		"raisonnable":        {demande: 50, attendu: 50},
		"au plafond":         {demande: pagination.MaxLimit, attendu: pagination.MaxLimit},
		"au-delà du plafond": {demande: 1_000_000, attendu: pagination.MaxLimit},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := pagination.NewRequest("", tc.demande)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if req.Limit != tc.attendu {
				t.Errorf("limite = %d, attendu %d", req.Limit, tc.attendu)
			}
		})
	}
}
