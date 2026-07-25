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

// Aides partagées par les tests du DÉPILEUR. Les aides du module lui-même sont
// dans helpers_test.go.

// dispatchClock avance d'un pas fixe à chaque lecture : les durées mesurées sont
// donc déterministes, et aucun test n'attend.
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

// spy enregistre ce que le dépileur a décidé.
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

// events rend la suite des événements rapportés.
func (s *spy) events() []domain.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Event, 0, len(s.outcomes))
	for _, outcome := range s.outcomes {
		out = append(out, outcome.Event)
	}
	return out
}

// lastOutcome rend le dernier compte rendu.
func (s *spy) lastOutcome(t *testing.T) domain.Outcome {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.outcomes) == 0 {
		t.Fatal("aucun compte rendu")
	}
	return s.outcomes[len(s.outcomes)-1]
}

// lastAttempt rend le dernier échec enregistré.
func (s *spy) lastAttempt(t *testing.T) domain.FailedAttempt {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.failed) == 0 {
		t.Fatal("aucun échec enregistré")
	}
	return s.failed[len(s.failed)-1]
}

// claimOnce rend le lot donné au premier appel, puis plus rien.
//
// Sans cet épuisement, le rattrapage du dépileur boucle jusqu'à sa borne et le
// test compterait dix fois le même message.
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

// pending forge un message en attente.
func pending(id string, attempts int) domain.Message {
	return domain.Message{
		ID:       domain.MessageID(id),
		Type:     "user.registered",
		Payload:  []byte(`{}`),
		Status:   domain.StatusPending,
		Attempts: attempts,
	}
}

// testPolicy est une politique valide et lisible.
func testPolicy() application.Policy {
	return application.Policy{
		BatchSize:   10,
		MaxAttempts: 3,
		BaseBackoff: time.Second,
		Interval:    time.Millisecond,
	}
}

// newDispatcher construit un dépileur sur des closures.
func newDispatcher(t *testing.T, p application.Ports, policy application.Policy) *application.Dispatcher {
	t.Helper()
	dispatcher, err := application.NewDispatcher(p, policy)
	if err != nil {
		t.Fatalf("construction du dépileur: %v", err)
	}
	return dispatcher
}

// dispatcherPorts assemble des ports complets autour d'un publieur donné.
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
