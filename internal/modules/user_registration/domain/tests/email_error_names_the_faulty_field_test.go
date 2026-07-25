package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEmailErrorNamesTheFaultyField : une erreur de validation nomme son champ.
//
// Sans `Field`, une surface HTTP ne peut pas placer le message sous le bon
// formulaire : elle affiche une bannière générique et l'utilisateur cherche
// lui-même ce qui cloche. Le champ est ce qui transforme une erreur en correction
// possible.
func TestEmailErrorNamesTheFaultyField(t *testing.T) {
	t.Parallel()

	err := erreurDe(t, domain.NewEmail("pas une adresse"))
	if err.Field != "email" {
		t.Errorf("champ = %q, attendu \"email\"", err.Field)
	}
	if err.Message == "" {
		t.Error("le message destiné à l'utilisateur ne doit pas être vide")
	}
}
