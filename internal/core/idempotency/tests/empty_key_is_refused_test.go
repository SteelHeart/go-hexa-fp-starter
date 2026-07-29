package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestEmptyKeyIsRefused is the most important test of the module.
//
// An empty key would be shared by ALL callers: the first replay from anyone
// would return the memorised response of anyone else. A protection against
// duplicates would turn into a data leak between users.
func TestEmptyKeyIsRefused(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	cases := map[string]domain.Request{
		"empty key":                 {Key: "", Fingerprint: "abc"},
		"empty fingerprint":         {Key: "k1", Fingerprint: ""},
		"both empty":                {},
		"empty key, fingerprint ok": {Key: "", Fingerprint: domain.Fingerprint("payload")},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrIncomplete) {
				t.Errorf("Reserve = %v, want ErrIncomplete", err)
			}
		})
	}
}
