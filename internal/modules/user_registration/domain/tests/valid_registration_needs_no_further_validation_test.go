package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestValidRegistrationNeedsNoFurtherValidation : une commande valide produit un
// type qui PORTE sa validité.
//
// C'est le franchissement de frontière, et il n'a lieu qu'une fois. En aval,
// aucune étape ne revalide : elle ne pourrait pas, puisqu'elle ne reçoit plus de
// `string` mais des types du domaine. C'est ce qui élimine la validation défensive
// répétée — celle qu'on ajoute « au cas où » et qui finit par diverger d'un endroit
// à l'autre.
func TestValidRegistrationNeedsNoFurtherValidation(t *testing.T) {
	t.Parallel()

	cmd := domain.RegistrationCommand{
		Email:    "  Alice@Example.COM ",
		Password: "correct cheval batterie agrafe",
	}

	valide, err, ok := domain.ParseRegistration(cmd).Get()
	if !ok {
		t.Fatalf("commande valide refusée: %v", err)
	}
	if got := valide.Email.String(); got != "alice@example.com" {
		t.Errorf("adresse = %q, attendu la forme normalisée", got)
	}
	if valide.Password.Expose() != cmd.Password {
		t.Error("le mot de passe doit traverser la validation sans être altéré")
	}
}
