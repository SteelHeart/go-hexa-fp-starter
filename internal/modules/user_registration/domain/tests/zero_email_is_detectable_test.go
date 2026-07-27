package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestZeroEmailIsDetectable : une adresse non construite se reconnaît.
//
// Le type interdit de FABRIQUER une adresse invalide, mais pas d'en déclarer une
// vide — `var e Email` reste légal en Go. `IsZero` est ce qui permet à un
// adaptateur de refuser une valeur qui n'est jamais passée par `NewEmail`, plutôt
// que d'écrire une chaîne vide en base.
func TestZeroEmailIsDetectable(t *testing.T) {
	t.Parallel()

	var jamaisConstruite domain.Email
	if !jamaisConstruite.IsZero() {
		t.Error("une adresse non construite doit être détectable")
	}
	if jamaisConstruite.String() != "" {
		t.Errorf("adresse non construite = %q, attendu vide", jamaisConstruite.String())
	}

	if emailValide(t, "alice@example.com").IsZero() {
		t.Error("une adresse construite ne doit pas être vue comme vide")
	}
}
