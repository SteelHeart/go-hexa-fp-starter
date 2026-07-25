package tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestObjectWithoutContentIsRefused : un objet sans nom ou sans flux est refusé
// avant toute écriture.
//
// Un `Content` nil ferait paniquer io.Copy dans le pilote : le refus explicite
// transforme un plantage de processus en erreur que l'appelant peut traiter.
func TestObjectWithoutContentIsRefused(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	cases := map[string]domain.Object{
		"sans flux": {Name: "doc.pdf", Content: nil},
		"sans nom":  {Name: "", Content: strings.NewReader("x")},
		"vide":      {},
	}

	for name, obj := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := mod.Put(ctx, obj); !errors.Is(err, domain.ErrEmptyContent) {
				t.Errorf("Put = %v, attendu ErrEmptyContent", err)
			}
		})
	}
}
