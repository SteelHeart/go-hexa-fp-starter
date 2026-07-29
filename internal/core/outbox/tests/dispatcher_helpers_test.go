package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Helpers shared by the DISPATCHER tests. The helpers of the module itself are
// in helpers_test.go.

// dispatchClock advances by a fixed step at each read: the measured durations
// are therefore deterministic, and no test waits.
type dispatchClock struct {
	mu   sync.Mutex
	at   time.Time
	step time.Duration
}

func newDispatchClock() *dispatchClock {
	return &dispatchClock{at: fixedNow(), step: 10 * time.Millisecond}
}

func (c *dispatchClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.at
	c.at = c.at.Add(c.step)
	return current
}

// spy records what the dispatcher decided.
type spy struct {
	mu       sync.Mutex
	outcomes []domain.Outcome
	done     []domain.MessageID
	failed   []domain.FailedAttempt
}

func (s *spy) report() ports.Report {
	return func(_ context.Context, outcome domain.Outcome) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.outcomes = append(s.outcomes, outcome)
	}
}

func (s *spy) markDone() ports.MarkDone {
	return func(_ context.Context, id domain.MessageID) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.done = append(s.done, id)
		return nil
	}
}

func (s *spy) markFailed() ports.MarkFailed {
	return func(_ context.Context, attempt domain.FailedAttempt) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.failed = append(s.failed, attempt)
		return nil
	}
}

// lastOutcome returns the last report.
func (s *spy) lastOutcome(t *testing.T) domain.Outcome {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.outcomes) == 0 {
		t.Fatal("no report at all")
	}
	return s.outcomes[len(s.outcomes)-1]
}

// lastAttempt returns the last recorded failure.
func (s *spy) lastAttempt(t *testing.T) domain.FailedAttempt {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.failed) == 0 {
		t.Fatal("no recorded failure at all")
	}
	return s.failed[len(s.failed)-1]
}

// claimOnce returns the given batch on the first call, then nothing more.
//
// Without this exhaustion, the dispatcher's catch-up loops up to its bound and
// the test would count the same message ten times.
func claimOnce(messages ...domain.Message) ports.Claim {
	var served bool
	var mu sync.Mutex
	return func(context.Context, int) ([]domain.Message, error) {
		mu.Lock()
		defer mu.Unlock()
		if served {
			return nil, nil
		}
		served = true
		return messages, nil
	}
}

// pending forges a pending message.
func pending(id string, attempts int) domain.Message {
	return domain.Message{
		ID:       domain.MessageID(id),
		Type:     "user.registered",
		Payload:  []byte(`{}`),
		Status:   domain.StatusPending,
		Attempts: attempts,
	}
}

// testPolicy is a valid and readable policy.
func testPolicy() application.Policy {
	return application.Policy{
		BatchSize: 10,
		Interval:  time.Millisecond,
		Retry:     domain.RetryPolicy{MaxAttempts: 3, BaseBackoff: time.Second},
	}
}

// newDispatcher builds a dispatcher on top of closures.
func newDispatcher(t *testing.T, p application.Ports, policy application.Policy) *application.Dispatcher {
	t.Helper()
	dispatcher, err := application.NewDispatcher(p, policy)
	if err != nil {
		t.Fatalf("building the dispatcher: %v", err)
	}
	return dispatcher
}

// dispatcherPorts assembles complete ports around a given publisher.
func dispatcherPorts(s *spy, claim ports.Claim, handle ports.Handler) application.Ports {
	return application.Ports{
		Claim:      claim,
		MarkDone:   s.markDone(),
		MarkFailed: s.markFailed(),
		Handle:     handle,
		Report:     s.report(),
		Now:        newDispatchClock().now,
	}
}
