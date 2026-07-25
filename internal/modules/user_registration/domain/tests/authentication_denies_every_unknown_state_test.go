package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestAuthenticationDeniesEveryUnknownState : seul `active` autorise la connexion.
//
// Le `default` du switch refuse. Ce n'est pas de la paranoïa : le jour où quelqu'un
// ajoute un état — `suspended`, `deleted` — sans compléter la fonction, le
// comportement par défaut doit être le REFUS. L'inverse ouvrirait un compte
// suspendu à cause d'un oubli.
//
// Le linter `exhaustive` fait échouer la CI dans ce cas ; ce test couvre la
// situation où quelqu'un désactiverait le linter, ou construirait un état à la main.
func TestAuthenticationDeniesEveryUnknownState(t *testing.T) {
	t.Parallel()

	base := domain.NewUser(
		"user-42",
		emailValide(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	cases := map[domain.Status]bool{
		domain.StatusActive:  true,
		domain.StatusPending: false,
		domain.StatusBlocked: false,
		"état inventé":       false,
		"":                   false,
	}

	for status, attendu := range cases {
		if got := base.WithStatus(status).CanAuthenticate(); got != attendu {
			t.Errorf("CanAuthenticate avec l'état %q = %v, attendu %v", status, got, attendu)
		}
	}
}
