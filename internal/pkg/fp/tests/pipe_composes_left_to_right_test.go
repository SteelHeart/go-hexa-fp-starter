package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestPipeComposesLeftToRight : Pipe2 et Pipe3 composent dans l'ordre de LECTURE.
//
// L'ordre n'est pas un détail de goût. La composition mathématique s'écrit de
// droite à gauche — `g ∘ f` applique f d'abord — et c'est l'inverse de ce qu'un
// lecteur suppose. Le dépôt choisit l'ordre de lecture, et le vérifie, parce
// qu'une composition inversée produit un résultat plausible : `versTexte(double(3))`
// et `double(versTexte(3))` ne compilent pas tous les deux, mais deux
// transformations de même type, si.
func TestPipeComposesLeftToRight(t *testing.T) {
	t.Parallel()

	// double PUIS incremente : 3 → 6 → 7. L'ordre inverse donnerait 8.
	deux := fp.Pipe2(double, incremente)
	if got := deux(3); got != 7 {
		t.Errorf("Pipe2 = %d, attendu 7 (double puis incrémente)", got)
	}

	// double PUIS incremente PUIS versTexte : 3 → 6 → 7 → "7".
	trois := fp.Pipe3(double, incremente, versTexte)
	if got := trois(3); got != "7" {
		t.Errorf("Pipe3 = %q, attendu \"7\"", got)
	}
}
