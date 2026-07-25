package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestDispatcherRefusesMissingPorts : un port nil refuse à la construction.
//
// Sans ce refus, le nil ne se manifesterait qu'au premier tick, dans une goroutine,
// sous forme de panique — donc en emportant tout le processus, plusieurs secondes
// après un démarrage qui semblait réussi. Le pire moment pour l'apprendre.
func TestDispatcherRefusesMissingPorts(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	complete := dispatcherPorts(observed, claimOnce(),
		func(context.Context, domain.Message) error { return nil })

	cases := map[string]func(application.Ports) application.Ports{
		"sans Claim":      func(p application.Ports) application.Ports { p.Claim = nil; return p },
		"sans MarkDone":   func(p application.Ports) application.Ports { p.MarkDone = nil; return p },
		"sans MarkFailed": func(p application.Ports) application.Ports { p.MarkFailed = nil; return p },
		"sans Handle":     func(p application.Ports) application.Ports { p.Handle = nil; return p },
		"sans Report":     func(p application.Ports) application.Ports { p.Report = nil; return p },
		"sans Now":        func(p application.Ports) application.Ports { p.Now = nil; return p },
		"aucun port":      func(application.Ports) application.Ports { return application.Ports{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := application.NewDispatcher(mutate(complete), testPolicy())
			if !errors.Is(err, application.ErrMissingPort) {
				t.Errorf("NewDispatcher = %v, attendu ErrMissingPort", err)
			}
		})
	}
}
