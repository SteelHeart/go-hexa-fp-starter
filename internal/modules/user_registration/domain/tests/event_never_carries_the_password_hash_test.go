package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEventNeverCarriesThePasswordHash : l'événement porte le strict nécessaire.
//
// Y mettre l'utilisateur entier exposerait le condensé de mot de passe dans
// l'outbox — une table lue par des humains en incident, répliquée vers un courtier,
// et souvent conservée bien plus longtemps que la donnée d'origine.
//
// Un condensé Argon2id n'est pas un mot de passe, mais c'est un matériau
// d'attaque hors ligne : il n'a rien à faire dans un message.
func TestEventNeverCarriesThePasswordHash(t *testing.T) {
	t.Parallel()

	const condense = "$argon2id$v=19$m=65536,t=3,p=2$c2VjcmV0$Y29uZGVuc2U"
	user := domain.NewUser(
		"user-42",
		emailValide(t, "alice@example.com"),
		domain.NewPasswordHash(condense),
		time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC),
	)

	event := domain.NewUserRegistered(user)
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("sérialisation de l'événement: %v", err)
	}

	if strings.Contains(string(encoded), condense) {
		t.Errorf("l'événement transporte le condensé de mot de passe: %s", encoded)
	}
	if !strings.Contains(string(encoded), "user-42") {
		t.Error("l'événement doit porter l'identifiant : sans lui, il est inexploitable")
	}
}
