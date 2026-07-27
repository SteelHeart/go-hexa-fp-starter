package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestErrorBuildersReturnCopies : `WithField` et `WithCause` rendent une nouvelle
// erreur.
//
// Une Error est une VALEUR, pas une interface, précisément pour que l'ensemble des
// erreurs reste énumérable. Si les constructeurs mutaient leur receveur, une erreur
// sentinelle partagée — déclarée une fois au niveau paquet — se retrouverait
// enrichie du champ et de la cause du dernier appel. Deux requêtes concurrentes
// s'échangeraient alors leurs détails techniques.
func TestErrorBuildersReturnCopies(t *testing.T) {
	t.Parallel()

	base := domain.NewError(domain.CodeInternal, "erreur interne")

	avecChamp := base.WithField("email")
	avecCause := base.WithCause(errors.New("détail"))

	if base.Field != "" {
		t.Errorf("WithField a muté l'original: champ = %q", base.Field)
	}
	if base.Cause() != nil {
		t.Error("WithCause a muté l'original")
	}
	if avecChamp.Field != "email" {
		t.Errorf("la copie porte %q, attendu \"email\"", avecChamp.Field)
	}
	if avecCause.Cause() == nil {
		t.Error("la copie doit porter la cause")
	}
	if avecChamp.Cause() != nil {
		t.Error("les deux copies doivent être indépendantes l'une de l'autre")
	}
}
