package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestFetchLimitAsksForOneExtraRow : on demande UNE ligne de plus que nécessaire.
//
// C'est ce qui permet de savoir s'il existe une page suivante sans exécuter de
// `COUNT(*)`. Sur une grande table, ce COUNT coûte un parcours complet à chaque
// page — plus cher que la page elle-même, pour une information dont on n'a besoin
// que sous forme de booléen.
func TestFetchLimitAsksForOneExtraRow(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 20)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got := req.FetchLimit(); got != 21 {
		t.Errorf("FetchLimit = %d, attendu 21 (limite + 1)", got)
	}
}
