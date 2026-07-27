package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestPageDetectsMoreWithoutLeakingTheExtraRow : la ligne supplémentaire sert à
// détecter la suite, elle n'est JAMAIS rendue.
//
// C'est le défaut le plus facile à écrire ici : oublier la troncature ferait rendre
// vingt et une lignes pour une limite de vingt. Le client afficherait un élément de
// trop par page, et le retrouverait en tête de la page suivante.
func TestPageDetectsMoreWithoutLeakingTheExtraRow(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// FetchLimit vaut 4 : la base rend une ligne de plus que la limite.
	page := pagination.NewPage(lignes(4), req, curseurDe)

	if !page.HasMore {
		t.Error("HasMore doit être vrai quand la ligne supplémentaire existe")
	}
	if len(page.Items) != 3 {
		t.Fatalf("éléments rendus = %d, attendu 3 — la ligne témoin a fuité", len(page.Items))
	}
	if page.Items[2].ID != "c" {
		t.Errorf("dernier élément = %q, attendu \"c\"", page.Items[2].ID)
	}
	if page.NextCursor == "" {
		t.Error("une page suivante existe : NextCursor doit être renseigné")
	}
}
