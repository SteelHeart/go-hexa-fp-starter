package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// TestIncompleteEntryIsRefused : une entrée d'audit incomplète est pire
// qu'absente. Elle donne l'illusion d'une traçabilité — on croit pouvoir répondre
// à « qui a fait ça » jusqu'au jour où on essaie.
//
// Le refus est déclaré dans le DOMAINE, donc identique quel que soit le pilote
// branché, et reconnaissable par errors.Is. C'est la forme opérationnelle de la
// substituabilité (ADR 003).
func TestIncompleteEntryIsRefused(t *testing.T) {
	t.Parallel()

	mod, _ := newLogModule(t)
	ctx := context.Background()

	cases := map[string]func(domain.Entry) domain.Entry{
		"sans acteur": func(e domain.Entry) domain.Entry { e.Actor = ""; return e },
		"sans action": func(e domain.Entry) domain.Entry { e.Action = ""; return e },
		"sans type":   func(e domain.Entry) domain.Entry { e.EntityType = ""; return e },
		"sans id":     func(e domain.Entry) domain.Entry { e.EntityID = ""; return e },
		"entrée vide": func(domain.Entry) domain.Entry { return domain.Entry{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := mod.Record(ctx, mutate(completeEntry())); !errors.Is(err, domain.ErrIncomplete) {
				t.Errorf("Record = %v, attendu ErrIncomplete", err)
			}
		})
	}
}
