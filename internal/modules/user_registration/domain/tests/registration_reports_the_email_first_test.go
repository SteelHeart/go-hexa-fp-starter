package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestRegistrationReportsTheEmailFirst : quand l'adresse ET le mot de passe sont
// mauvais, c'est l'adresse qui est signalée.
//
// L'ordre n'est pas arbitraire : c'est le champ que l'utilisateur remplit en
// premier, donc celui qu'il corrigera en premier. Signaler le mot de passe d'abord
// le ferait corriger un champ pour découvrir ensuite que le précédent était fautif —
// deux allers-retours au lieu d'un.
//
// C'est une décision d'ergonomie, prise une fois, et ce test l'empêche de se
// perdre au premier refactoring du pipeline de validation.
func TestRegistrationReportsTheEmailFirst(t *testing.T) {
	t.Parallel()

	lesDeuxMauvais := domain.RegistrationCommand{Email: "pas une adresse", Password: "court"}
	if got := codeDe(t, domain.ParseRegistration(lesDeuxMauvais)); got != domain.CodeInvalidEmail {
		t.Errorf("code = %q, attendu %q en premier", got, domain.CodeInvalidEmail)
	}

	adresseValide := domain.RegistrationCommand{Email: "alice@example.com", Password: "court"}
	if got := codeDe(t, domain.ParseRegistration(adresseValide)); got != domain.CodeWeakPassword {
		t.Errorf("code = %q, attendu %q une fois l'adresse valide", got, domain.CodeWeakPassword)
	}
}
