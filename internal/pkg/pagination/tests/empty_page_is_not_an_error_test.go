package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestEmptyPageIsNotAnError : zéro résultat est une réponse valide.
//
// Une recherche sans correspondance, une liste jamais alimentée, une page demandée
// juste après la dernière : aucun de ces cas n'est une panne. Le seul piège serait
// de rendre un `NextCursor` sur une page vide — le client boucherait.
func TestEmptyPageIsNotAnError(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 10)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	page := pagination.NewPage([]ligne{}, req, curseurDe)
	if page.HasMore {
		t.Error("une page vide n'a pas de suite")
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor = %q, attendu vide", page.NextCursor)
	}
	if len(page.Items) != 0 {
		t.Errorf("éléments = %d, attendu 0", len(page.Items))
	}
}
