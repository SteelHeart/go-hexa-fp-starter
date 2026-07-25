package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestInvalidPolicyRefusesConstruction : une politique absurde refuse au démarrage.
//
// Chaque cas décrit un comportement silencieusement faux, pas une simple faute de
// saisie :
//   - lot nul : le dépileur tourne sans jamais rien traiter, et le retard grossit
//     sans qu'aucune erreur n'apparaisse ;
//   - zéro tentative : tout message passe en `failed` sans qu'une seule publication
//     ait été tentée ;
//   - recul nul : un message en échec est rejoué sans pause, en boucle serrée ;
//   - période nulle : `time.NewTicker(0)` PANIQUE, dans une goroutine de worker,
//     donc emporte tout le processus.
func TestInvalidPolicyRefusesConstruction(t *testing.T) {
	t.Parallel()

	valid := application.Policy{
		BatchSize: 10,
		Interval:  time.Second,
		Retry:     domain.RetryPolicy{MaxAttempts: 3, BaseBackoff: time.Second},
	}

	cases := map[string]func(application.Policy) application.Policy{
		"lot nul":          func(p application.Policy) application.Policy { p.BatchSize = 0; return p },
		"lot négatif":      func(p application.Policy) application.Policy { p.BatchSize = -1; return p },
		"zéro tentative":   func(p application.Policy) application.Policy { p.Retry.MaxAttempts = 0; return p },
		"recul nul":        func(p application.Policy) application.Policy { p.Retry.BaseBackoff = 0; return p },
		"recul négatif":    func(p application.Policy) application.Policy { p.Retry.BaseBackoff = -time.Second; return p },
		"période nulle":    func(p application.Policy) application.Policy { p.Interval = 0; return p },
		"période négative": func(p application.Policy) application.Policy { p.Interval = -time.Second; return p },
	}

	observed := &spy{}
	base := dispatcherPorts(observed, claimOnce(), func(context.Context, domain.Message) error { return nil })

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := application.NewDispatcher(base, mutate(valid))
			if !errors.Is(err, application.ErrInvalidPolicy) {
				t.Errorf("NewDispatcher = %v, attendu ErrInvalidPolicy", err)
			}
		})
	}
}
