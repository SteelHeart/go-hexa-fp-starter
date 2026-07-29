package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestDispatcherRefusesMissingPorts: a nil port is refused at construction.
//
// Without this refusal, the nil would only show up at the first tick, in a
// goroutine, as a panic — hence taking down the whole process, several seconds
// after a startup that looked successful. The worst moment to learn about it.
func TestDispatcherRefusesMissingPorts(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	complete := dispatcherPorts(observed, claimOnce(),
		func(context.Context, domain.Message) error { return nil })

	cases := map[string]func(application.Ports) application.Ports{
		"without Claim":      func(p application.Ports) application.Ports { p.Claim = nil; return p },
		"without MarkDone":   func(p application.Ports) application.Ports { p.MarkDone = nil; return p },
		"without MarkFailed": func(p application.Ports) application.Ports { p.MarkFailed = nil; return p },
		"without Handle":     func(p application.Ports) application.Ports { p.Handle = nil; return p },
		"without Report":     func(p application.Ports) application.Ports { p.Report = nil; return p },
		"without Now":        func(p application.Ports) application.Ports { p.Now = nil; return p },
		"no port at all":     func(application.Ports) application.Ports { return application.Ports{} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := application.NewDispatcher(mutate(complete), testPolicy())
			if !errors.Is(err, application.ErrMissingPort) {
				t.Errorf("NewDispatcher = %v, want ErrMissingPort", err)
			}
		})
	}
}
