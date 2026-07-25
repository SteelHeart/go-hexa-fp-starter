package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestLongPassphraseIsAccepted : une phrase de passe ordinaire passe, sans exiger
// de majuscule, de chiffre ni de caractère spécial.
//
// C'est la contrepartie du minimum élevé, et elle doit être testée : une règle de
// composition ajoutée « pour faire sérieux » rendrait ce test rouge, et c'est
// exactement le signal qu'on veut — la décision a été prise, elle ne se défait pas
// par inadvertance.
func TestLongPassphraseIsAccepted(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"correct cheval batterie agrafe",
		"jaime beaucoup les tartes aux pommes",
		"douzeCARACT1",
	} {
		if _, _, ok := domain.NewRawPassword(raw).Get(); !ok {
			t.Errorf("NewRawPassword(%q) devait être acceptée", raw)
		}
	}
}
