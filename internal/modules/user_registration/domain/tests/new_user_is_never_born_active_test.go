package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestNewUserIsNeverBornActive : un compte naît PENDING.
//
// Naître actif serait un fail-open : à cet instant, l'adresse n'est PAS prouvée.
// N'importe qui pourrait s'inscrire avec l'adresse d'un autre et disposer
// immédiatement d'un compte fonctionnel à son nom.
//
// C'est « deny par défaut » appliqué au cycle de vie, et c'est le genre de décision
// qu'une refonte du formulaire d'inscription défait sans s'en apercevoir.
func TestNewUserIsNeverBornActive(t *testing.T) {
	t.Parallel()

	user := domain.NewUser(
		"user-42",
		emailValide(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	if user.Status != domain.StatusPending {
		t.Errorf("état = %q, attendu %q", user.Status, domain.StatusPending)
	}
	if user.CanAuthenticate() {
		t.Error("un compte dont l'adresse n'est pas prouvée ne doit PAS pouvoir se connecter")
	}
}
