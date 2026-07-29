// Package memory implements the outbox in memory.
//
// # NON-GUARANTEES — read before using it
//
//   - **No durability.** Everything is lost when the process restarts.
//   - **No sharing between replicas.** Each instance has its own queue.
//   - **No atomicity with the business transaction.** A successful `Enqueue`
//     survives a database rollback — exactly the defect that the transactional
//     outbox exists to eliminate.
//
// It is therefore the right driver in development, in unit tests and for a CLI.
// It is NEVER the right driver in production as soon as an event matters.
//
// It is not a stub for all that: it honours the same contract as the `postgres`
// driver, including the exclusivity of `Claim`, and passes the same conformance
// suite.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// Store is the in-memory queue.
type Store struct {
	mu       sync.Mutex
	messages map[domain.MessageID]*domain.Message
	// claimed marks the messages claimed by a Claim not yet resolved, so that two
	// concurrent calls never return the same message.
	claimed map[domain.MessageID]struct{}
	order   []domain.MessageID
	nextID  uint64
	now     func() time.Time
}

// New builds the store.
//
// `now` is injected: a retry policy test must not wait.
func New(now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{
		messages: make(map[domain.MessageID]*domain.Message),
		claimed:  make(map[domain.MessageID]struct{}),
		now:      now,
	}
}

// Enqueue implements ports.Enqueue.
func (s *Store) Enqueue(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("enqueueing cancelled: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	id := domain.MessageID(fmt.Sprintf("mem-%d", s.nextID))
	now := s.now()
	s.messages[id] = &domain.Message{
		ID:          id,
		Type:        msg.Type,
		AggregateID: msg.AggregateID,
		Payload:     msg.Payload,
		TraceParent: msg.TraceParent,
		Headers:     msg.Headers,
		Status:      domain.StatusPending,
		CreatedAt:   now,
		AvailableAt: now,
	}
	s.order = append(s.order, id)
	return id, nil
}

// Claim implements ports.Claim.
//
// The order follows insertion, like the `ORDER BY available_at` of the postgres
// driver.
func (s *Store) Claim(ctx context.Context, limit int) ([]domain.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("claiming cancelled: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	out := make([]domain.Message, 0, limit)
	for _, id := range s.order {
		if len(out) >= limit {
			break
		}
		msg, found := s.messages[id]
		if !found || !msg.IsDue(now) {
			continue
		}
		if _, busy := s.claimed[id]; busy {
			continue
		}
		s.claimed[id] = struct{}{}
		out = append(out, *msg)
	}
	return out, nil
}

// MarkDone implements ports.MarkDone.
func (s *Store) MarkDone(_ context.Context, id domain.MessageID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, found := s.messages[id]
	if !found {
		return fmt.Errorf("unknown message %s", id)
	}
	msg.Status = domain.StatusDone
	delete(s.claimed, id)
	return nil
}

// MarkFailed implements ports.MarkFailed.
func (s *Store) MarkFailed(_ context.Context, attempt domain.FailedAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, found := s.messages[attempt.ID]
	if !found {
		return fmt.Errorf("unknown message %s", attempt.ID)
	}
	msg.Attempts = attempt.Attempts
	msg.Status = attempt.Status
	msg.AvailableAt = attempt.AvailableAt
	delete(s.claimed, attempt.ID)
	return nil
}

// PendingCount implements ports.PendingCount.
func (s *Store) PendingCount(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for _, msg := range s.messages {
		if msg.Status == domain.StatusPending {
			count++
		}
	}
	return count, nil
}

// The store does NOT expose a method that would return its five ports at once:
// `revive: function-result-limit` would refuse it, and rightly so. It is
// module.go that assembles the Module struct field by field.
