package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestEmptyCursorMeansFirstPage : un curseur vide vaut « première page », ce n'est
// pas une erreur.
//
// `HasAfter` distingue « pas de curseur » de « curseur à la valeur zéro ». Sans ce
// booléen, la première page serait indistinguable d'une demande partant du 1ᵉʳ
// janvier de l'an 1 — et la requête SQL comporterait un `WHERE created_at > …`
// inutile, donc un index parcouru pour rien.
func TestEmptyCursorMeansFirstPage(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 10)
	if err != nil {
		t.Fatalf("un curseur vide ne doit pas être une erreur: %v", err)
	}
	if req.HasAfter {
		t.Error("HasAfter doit être faux sur la première page")
	}

	suivante, err := pagination.NewRequest(
		pagination.Cursor{CreatedAt: base(), ID: "a"}.Encode(), 10)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if !suivante.HasAfter {
		t.Error("HasAfter doit être vrai quand un curseur est fourni")
	}
	if suivante.After.ID != "a" {
		t.Errorf("curseur repris = %q, attendu \"a\"", suivante.After.ID)
	}
}
