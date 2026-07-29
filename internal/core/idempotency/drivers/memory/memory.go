// Package memory implements idempotency in memory.
//
// # NON-GUARANTEES — read before using it
//
//   - **No sharing across replicas.** Two instances behind a load balancer do
//     not see each other: the same request replayed on the other instance will
//     execute twice. It is the very guarantee of the module that disappears.
//   - **No durability.** Everything is lost on restart: a replay arriving just
//     after a deployment will execute a second time.
//   - **No passive expiry.** Keys only disappear when `Purge` runs: without a
//     scheduler, the map grows without bound.
//
// Suitable in development, in unit tests, for a CLI and for any single-instance
// binary. NOT suitable as soon as there are two replicas.
//
// It is not a stub for all that: the exclusivity of `Reserve` is genuinely held
// within the process, and the driver passes the same conformance suite as the
// `postgres` driver.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// entry is a live or completed reservation.
type entry struct {
	fingerprint string
	status      domain.Status
	response    []byte
	expiresAt   time.Time
}

// Store is the in-memory store.
type Store struct {
	mu      sync.Mutex
	entries map[domain.Key]*entry
	ttl     time.Duration
	now     func() time.Time
}

// New builds the store.
//
// `now` is injected: an expiry test must not wait.
func New(ttl time.Duration, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{entries: make(map[domain.Key]*entry), ttl: ttl, now: now}
}

// Reserve implements ports.Reserve.
//
// Exclusivity comes from the mutex held throughout the decision: reading the
// existing reservation and writing the new one are indivisible. Releasing the
// lock in between would let two concurrent calls obtain the key.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrIncomplete, req.Key)
	}
	if err := ctx.Err(); err != nil {
		return domain.Reservation{}, fmt.Errorf("reservation cancelled: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if held, found := s.entries[req.Key]; found && held.expiresAt.After(now) {
		return decide(*held, req)
	}
	s.entries[req.Key] = &entry{
		fingerprint: req.Fingerprint,
		status:      domain.StatusInFlight,
		expiresAt:   now.Add(s.ttl),
	}
	return domain.Reservation{}, nil
}

// decide settles the fate of a live reservation — outcomes 3, 4 and 5 of the
// ports.Reserve contract.
func decide(held entry, req domain.Request) (domain.Reservation, error) {
	if held.fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrConflict, req.Key)
	}
	if held.status == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: clone(held.response)}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrInFlight, req.Key)
}

// Complete implements ports.Complete.
func (s *Store) Complete(_ context.Context, key domain.Key, response []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	held, found := s.entries[key]
	if !found || !held.expiresAt.After(s.now()) {
		return fmt.Errorf("%w: key=%q", domain.ErrNotReserved, key)
	}
	held.status = domain.StatusDone
	held.response = clone(response)
	return nil
}

// Release implements ports.Release.
//
// A completed key is never freed: that would reopen the door to the replay.
func (s *Store) Release(_ context.Context, key domain.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if held, found := s.entries[key]; found && held.status == domain.StatusInFlight {
		delete(s.entries, key)
	}
	return nil
}

// Purge implements ports.Purge.
func (s *Store) Purge(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	var removed int64
	for key, held := range s.entries {
		if !held.expiresAt.After(now) {
			delete(s.entries, key)
			removed++
		}
	}
	return removed, nil
}

// clone copies the payload.
//
// Without this copy, the caller would keep a reference on the store's memory and
// could alter a memorised response — a replay would then return something other
// than the first call, silently.
func clone(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
