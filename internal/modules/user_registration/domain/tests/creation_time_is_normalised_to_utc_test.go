package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestCreationTimeIsNormalisedToUTC : la date de création est stockée en UTC.
//
// Un horodatage porteur d'un fuseau se compare mal, se trie mal, et se relit six
// mois plus tard depuis un autre pays. Pire : un serveur déplacé d'une région à
// l'autre produirait des dates qui semblent remonter le temps par rapport aux
// précédentes.
//
// La normalisation a lieu à la CONSTRUCTION, une fois, plutôt qu'à chaque lecture.
func TestCreationTimeIsNormalisedToUTC(t *testing.T) {
	t.Parallel()

	local := time.FixedZone("UTC+3", 3*60*60)
	at := time.Date(2026, time.July, 25, 15, 30, 0, 0, local)

	user := domain.NewUser(
		"user-42",
		emailValide(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		at,
	)

	if user.CreatedAt.Location() != time.UTC {
		t.Errorf("fuseau = %v, attendu UTC", user.CreatedAt.Location())
	}
	if !user.CreatedAt.Equal(at) {
		t.Errorf("l'instant a été déplacé: %v ≠ %v", user.CreatedAt, at)
	}
}
