package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestInvalidEmailIsRefused : le type Email ne peut PAS porter une valeur invalide.
//
// Le champ est non exporté et `NewEmail` est le seul chemin de construction :
// toute fonction qui reçoit un Email n'a donc plus à le valider. C'est le type qui
// le garantit, pas une convention de revue — et ce test est ce qui rend cette
// garantie vraie.
//
// Le cas « Nom <a@b.c> » mérite une mention : `mail.ParseAddress` l'accepte
// volontiers. Sans le refus explicite, deux saisies différentes donneraient le même
// utilisateur, et le nom affiché viendrait d'une entrée non validée.
func TestInvalidEmailIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"vide":                   "",
		"espaces seuls":          "   ",
		"sans arobase":           "alice.example.com",
		"sans domaine":           "alice@",
		"sans partie locale":     "@example.com",
		"domaine sans point":     "alice@localhost",
		"forme avec nom affiché": "Alice <alice@example.com>",
		"deux arobases":          "alice@@example.com",
		"espace intérieur":       "ali ce@example.com",
		"trop longue":            strings.Repeat("a", 250) + "@example.com",
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := codeDe(t, domain.NewEmail(raw)); got != domain.CodeInvalidEmail {
				t.Errorf("code = %q, attendu %q", got, domain.CodeInvalidEmail)
			}
		})
	}
}
