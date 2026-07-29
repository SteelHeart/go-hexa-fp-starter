package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// TestIncompleteEntryIsRefused: an incomplete audit entry is worse than an
// absent one. It gives the illusion of traceability — one believes one can
// answer "who did that" until the day one tries.
//
// The refusal is declared in the DOMAIN, hence identical whichever driver is
// plugged in, and recognisable through errors.Is. This is the operational form
// of substitutability (ADR 003).
func TestIncompleteEntryIsRefused(t *testing.T) {
	t.Parallel()

	mod, _ := newLogModule(t)
	ctx := context.Background()

	cases := map[string]func(domain.Entry) domain.Entry{
		"without actor":  func(e domain.Entry) domain.Entry { e.Actor = ""; return e },
		"without action": func(e domain.Entry) domain.Entry { e.Action = ""; return e },
		"without type":   func(e domain.Entry) domain.Entry { e.EntityType = ""; return e },
		"without id":     func(e domain.Entry) domain.Entry { e.EntityID = ""; return e },
		"empty entry":    func(domain.Entry) domain.Entry { return domain.Entry{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := mod.Record(ctx, mutate(completeEntry())); !errors.Is(err, domain.ErrIncomplete) {
				t.Errorf("Record = %v, want ErrIncomplete", err)
			}
		})
	}
}
