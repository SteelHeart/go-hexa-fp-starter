package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestCipherRefusesAKeyOfWrongLength: a key that is not 32 bytes long refuses
// construction.
//
// The refusal must happen at STARTUP, not on the first piece of encrypted data.
// A badly sized key almost always comes from a truncated or badly encoded
// environment variable — the kind of defect you want to discover at deployment
// time, not the following night.
func TestCipherRefusesAKeyOfWrongLength(t *testing.T) {
	t.Parallel()

	for name, size := range map[string]int{
		"empty": 0, "too short": 16, "almost": 31, "too long": 64,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := security.NewCipher(make([]byte, size)); err == nil {
				t.Errorf("a %d-byte key must be refused", size)
			}
		})
	}

	if _, err := security.NewCipher(aesKey()); err != nil {
		t.Errorf("a 32-byte key must be accepted: %v", err)
	}
}
