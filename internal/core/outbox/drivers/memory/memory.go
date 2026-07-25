// Package memory implémente l'outbox en mémoire.
//
// # NON-GARANTIES — à lire avant de l'utiliser
//
//   - **Aucune durabilité.** Tout est perdu au redémarrage du processus.
//   - **Aucun partage entre répliques.** Chaque instance a sa propre file.
//   - **Aucune atomicité avec la transaction métier.** Un `Enqueue` réussi
//     survit à un rollback de la base — exactement le défaut que l'outbox
//     transactionnel existe pour éliminer.
//
// C'est donc le bon pilote en développement, en test unitaire et pour une CLI.
// Ce n'est JAMAIS le bon pilote en production dès qu'un événement compte.
//
// Il n'est pas un bouchon pour autant : il respecte le même contrat que le
// pilote `postgres`, y compris l'exclusivité de `Claim`, et passe la même suite
// de conformité.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// Store est la file en mémoire.
type Store struct {
	mu       sync.Mutex
	messages map[domain.MessageID]*domain.Message
	// claimed marque les messages réservés par un Claim non encore résolu, pour
	// que deux appels concurrents ne retournent jamais le même message.
	claimed map[domain.MessageID]struct{}
	order   []domain.MessageID
	nextID  uint64
	now     func() time.Time
}

// New construit le magasin.
//
// `now` est injecté : un test de politique de réessai ne doit pas attendre.
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

// Enqueue implémente ports.Enqueue.
func (s *Store) Enqueue(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("mise en file annulée: %w", err)
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

// Claim implémente ports.Claim.
//
// L'ordre suit l'insertion, comme le `ORDER BY available_at` du pilote postgres.
func (s *Store) Claim(ctx context.Context, limit int) ([]domain.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("réservation annulée: %w", err)
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

// MarkDone implémente ports.MarkDone.
func (s *Store) MarkDone(_ context.Context, id domain.MessageID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, found := s.messages[id]
	if !found {
		return fmt.Errorf("message %s inconnu", id)
	}
	msg.Status = domain.StatusDone
	delete(s.claimed, id)
	return nil
}

// MarkFailed implémente ports.MarkFailed.
func (s *Store) MarkFailed(_ context.Context, attempt domain.FailedAttempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, found := s.messages[attempt.ID]
	if !found {
		return fmt.Errorf("message %s inconnu", attempt.ID)
	}
	msg.Attempts = attempt.Attempts
	msg.Status = attempt.Status
	msg.AvailableAt = attempt.AvailableAt
	delete(s.claimed, attempt.ID)
	return nil
}

// PendingCount implémente ports.PendingCount.
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

// Le magasin n'expose PAS de méthode qui rendrait ses cinq ports d'un coup :
// `revive: function-result-limit` la refuserait, et à juste titre. C'est
// module.go qui assemble la struct Module champ par champ.
