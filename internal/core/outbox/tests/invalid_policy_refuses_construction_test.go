package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestInvalidPolicyRefusesConstruction: an absurd policy is refused at startup.
//
// Each case describes a silently wrong behaviour, not a mere typo:
//   - zero batch: the dispatcher runs without ever processing anything, and the
//     backlog grows without any error appearing;
//   - zero attempts: every message goes to `failed` without a single
//     publication having been attempted;
//   - zero backoff: a failed message is replayed without a pause, in a tight
//     loop;
//   - zero period: `time.NewTicker(0)` PANICS, in a worker goroutine, hence
//     takes down the whole process.
func TestInvalidPolicyRefusesConstruction(t *testing.T) {
	t.Parallel()

	valid := application.Policy{
		BatchSize: 10,
		Interval:  time.Second,
		Retry:     domain.RetryPolicy{MaxAttempts: 3, BaseBackoff: time.Second},
	}

	cases := map[string]func(application.Policy) application.Policy{
		"zero batch":       func(p application.Policy) application.Policy { p.BatchSize = 0; return p },
		"negative batch":   func(p application.Policy) application.Policy { p.BatchSize = -1; return p },
		"zero attempts":    func(p application.Policy) application.Policy { p.Retry.MaxAttempts = 0; return p },
		"zero backoff":     func(p application.Policy) application.Policy { p.Retry.BaseBackoff = 0; return p },
		"negative backoff": func(p application.Policy) application.Policy { p.Retry.BaseBackoff = -time.Second; return p },
		"zero period":      func(p application.Policy) application.Policy { p.Interval = 0; return p },
		"negative period":  func(p application.Policy) application.Policy { p.Interval = -time.Second; return p },
	}

	observed := &spy{}
	base := dispatcherPorts(observed, claimOnce(), func(context.Context, domain.Message) error { return nil })

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := application.NewDispatcher(base, mutate(valid))
			if !errors.Is(err, application.ErrInvalidPolicy) {
				t.Errorf("NewDispatcher = %v, want ErrInvalidPolicy", err)
			}
		})
	}
}
