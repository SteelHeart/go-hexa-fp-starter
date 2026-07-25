package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestEmptyKeyIsRefused est le test le plus important du module.
//
// Une clé vide serait partagée par TOUS les appelants : le premier rejeu de
// n'importe qui retournerait la réponse mémorisée de n'importe quel autre. Une
// protection contre les doublons se transformerait en fuite de données entre
// utilisateurs.
func TestEmptyKeyIsRefused(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	cases := map[string]domain.Request{
		"clé vide":               {Key: "", Fingerprint: "abc"},
		"empreinte vide":         {Key: "k1", Fingerprint: ""},
		"les deux vides":         {},
		"clé vide, empreinte ok": {Key: "", Fingerprint: domain.Fingerprint("charge")},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrIncomplete) {
				t.Errorf("Reserve = %v, attendu ErrIncomplete", err)
			}
		})
	}
}
