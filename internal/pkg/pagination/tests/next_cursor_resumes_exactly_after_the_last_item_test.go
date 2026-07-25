package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestNextCursorResumesExactlyAfterTheLastItem : le curseur suivant désigne le
// DERNIER élément rendu, pas la ligne témoin.
//
// C'est le cœur du contrat de pagination. S'il désignait la ligne supplémentaire —
// celle qu'on a lue mais pas rendue — la page suivante commencerait un cran trop
// loin et cet élément ne serait JAMAIS affiché. Le défaut est silencieux : rien ne
// signale une ligne manquante entre deux pages.
func TestNextCursorResumesExactlyAfterTheLastItem(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	page := pagination.NewPage(lignes(4), req, curseurDe)
	dernier := page.Items[len(page.Items)-1]

	repris, err := pagination.DecodeCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("le curseur rendu doit être décodable: %v", err)
	}
	if repris.ID != dernier.ID {
		t.Errorf("curseur suivant = %q, attendu le dernier élément RENDU (%q)",
			repris.ID, dernier.ID)
	}
	if !repris.CreatedAt.Equal(dernier.CreatedAt) {
		t.Errorf("horodatage du curseur = %v, attendu %v", repris.CreatedAt, dernier.CreatedAt)
	}
}
