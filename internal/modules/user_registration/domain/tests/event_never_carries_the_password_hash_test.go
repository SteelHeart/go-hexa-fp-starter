package tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEventNeverCarriesThePasswordHash: the event carries the strict minimum.
//
// Putting the whole user in it would expose the password digest in the outbox —
// a table read by humans during an incident, replicated towards a broker, and
// often kept much longer than the original data.
//
// An Argon2id digest is not a password, but it is offline attack material: it
// has no business being in a message.
func TestEventNeverCarriesThePasswordHash(t *testing.T) {
	t.Parallel()

	const digest = "$argon2id$v=19$m=65536,t=3,p=2$c2VjcmV0$Y29uZGVuc2U"
	user := domain.NewUser(
		"user-42",
		validEmail(t, "alice@example.com"),
		domain.NewPasswordHash(digest),
		time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC),
	)

	event := domain.NewUserRegistered(user)
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("serialising the event: %v", err)
	}

	if strings.Contains(string(encoded), digest) {
		t.Errorf("the event carries the password digest: %s", encoded)
	}
	if !strings.Contains(string(encoded), "user-42") {
		t.Error("the event must carry the identifier: without it, it is unusable")
	}
}
