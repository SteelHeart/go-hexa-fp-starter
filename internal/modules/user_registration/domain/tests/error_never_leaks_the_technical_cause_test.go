package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestErrorNeverLeaksTheTechnicalCause : le détail technique est journalisable,
// jamais retournable.
//
// Une erreur SQL renvoyée au client est une fuite de structure : noms de tables, de
// colonnes, de contraintes. C'est un plan du schéma offert, et l'une des premières
// choses que cherche un attaquant.
//
// La cause est donc portée dans un champ NON exporté, accessible par `Cause()` pour
// le journal, et absente de `Error()` que les surfaces affichent.
func TestErrorNeverLeaksTheTechnicalCause(t *testing.T) {
	t.Parallel()

	const secretTechnique = "pq: duplicate key value violates unique constraint \"users_email_key\""
	cause := errors.New(secretTechnique)

	err := domain.NewError(domain.CodeEmailAlreadyExists, "cette adresse est déjà enregistrée").
		WithField("email").
		WithCause(cause)

	if strings.Contains(err.Error(), secretTechnique) {
		t.Errorf("Error() laisse fuir le détail technique: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("Error() = %q : le champ fautif doit y figurer", err.Error())
	}
	if !errors.Is(err.Cause(), cause) {
		t.Error("Cause() doit rendre le détail technique, pour le journal")
	}
}
