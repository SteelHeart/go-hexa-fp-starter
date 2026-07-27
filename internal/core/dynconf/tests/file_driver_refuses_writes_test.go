package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestFileDriverRefusesWrites : écrire dans des valeurs versionnées à l'exécution
// créerait une divergence entre le dépôt et ce qui tourne, que le prochain
// déploiement écraserait sans prévenir. Le refus est explicite, jamais silencieux.
func TestFileDriverRefusesWrites(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, nil)
	ctx := context.Background()

	valide := domain.Change{Kind: domain.KindFlag, Key: "mode", Value: "true"}
	if err := mod.Set(ctx, valide); !errors.Is(err, domain.ErrReadOnly) {
		t.Errorf("Set = %v, attendu ErrReadOnly", err)
	}
}

// TestFileDriverReportsMalformedChangeFirst : une modification mal formée est
// signalée AVANT le refus d'écriture.
//
// L'ordre compte : l'appelant qui passera au pilote `postgres` pour pouvoir écrire
// doit d'abord apprendre que sa modification était fausse. Sinon il changerait de
// pilote et découvrirait l'erreur seulement là.
func TestFileDriverReportsMalformedChangeFirst(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, nil)
	ctx := context.Background()

	cases := map[string]domain.Change{
		"sans clé":         {Kind: domain.KindFlag},
		"nature inconnue":  {Kind: "autre", Key: "mode"},
		"nature manquante": {Key: "mode"},
	}

	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := mod.Set(ctx, change); !errors.Is(err, domain.ErrInvalidChange) {
				t.Errorf("Set = %v, attendu ErrInvalidChange", err)
			}
		})
	}
}
