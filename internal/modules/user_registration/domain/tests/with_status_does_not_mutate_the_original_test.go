package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestWithStatusDoesNotMutateTheOriginal : `WithStatus` rend une COPIE.
//
// L'immuabilité n'est pas un principe décoratif ici. Une valeur partagée entre un
// décorateur de cache, un journal d'audit et le pipeline métier serait modifiée
// pour tout le monde à la fois — et l'audit enregistrerait un état que
// l'utilisateur n'a jamais eu au moment du fait consigné.
func TestWithStatusDoesNotMutateTheOriginal(t *testing.T) {
	t.Parallel()

	original := domain.NewUser(
		"user-42",
		emailValide(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	active := original.WithStatus(domain.StatusActive)

	if original.Status != domain.StatusPending {
		t.Errorf("l'original a été muté: %q", original.Status)
	}
	if active.Status != domain.StatusActive {
		t.Errorf("la copie porte %q, attendu %q", active.Status, domain.StatusActive)
	}
	if active.ID != original.ID || active.Email.String() != original.Email.String() {
		t.Error("la copie doit conserver les autres champs à l'identique")
	}
}
