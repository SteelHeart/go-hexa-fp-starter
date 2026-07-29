package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestCreationTimeIsNormalisedToUTC: the creation date is stored in UTC.
//
// A timestamp carrying a time zone compares badly, sorts badly, and reads badly
// six months later from another country. Worse: a server moved from one region
// to another would produce dates that seem to go back in time compared with the
// previous ones.
//
// The normalisation happens at CONSTRUCTION, once, rather than on every read.
func TestCreationTimeIsNormalisedToUTC(t *testing.T) {
	t.Parallel()

	local := time.FixedZone("UTC+3", 3*60*60)
	at := time.Date(2026, time.July, 25, 15, 30, 0, 0, local)

	user := domain.NewUser(
		"user-42",
		validEmail(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		at,
	)

	if user.CreatedAt.Location() != time.UTC {
		t.Errorf("time zone = %v, want UTC", user.CreatedAt.Location())
	}
	if !user.CreatedAt.Equal(at) {
		t.Errorf("the instant was moved: %v != %v", user.CreatedAt, at)
	}
}
