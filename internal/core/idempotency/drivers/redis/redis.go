// Package redis implements idempotency on Redis.
//
// # GUARANTEES
//
//   - **Exclusivity across replicas**: `SET NX` is atomic on the server side.
//     Only one of the concurrent requests obtains the key.
//   - **Passive expiry**: Redis erases the keys on its own, so no purge is
//     needed and the store does not grow without bound.
//
// # NON-GUARANTEES
//
//   - **No atomicity with the business database.** Redis and the database are
//     two stores: a `Complete` may succeed while the business transaction has
//     failed. The caller must call `Release` on its failure path.
//   - **`Complete` is not atomic**: it reads the fingerprint then rewrites
//     (`SET XX`). An expiry slipping in between makes the memorisation fail,
//     never the business operation.
//   - **No strong durability.** A Redis without persistence, or flushed, loses
//     the memorisations: replays in flight will execute a second time.
//   - **Responses live in memory.** Memorising large payloads weighs on the
//     instance; keep this driver for short responses.
//
// # Status
//
// Written, NEVER run against a real Redis. Do not present it as exercised.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// envelope is what is stored under the key.
//
// A single record carries the fingerprint, the status and the response: the
// decision of `Reserve` is therefore taken on ONE read, with no partial state
// possible across several Redis keys.
type envelope struct {
	Fingerprint string        `json:"fingerprint"`
	Status      domain.Status `json:"status"`
	Response    []byte        `json:"response,omitempty"`
}

// Store implements idempotency on Redis.
type Store struct {
	client    *goredis.Client
	namespace string
	ttl       time.Duration
}

// New builds the store.
//
// `namespace` prefixes every key: two deployments sharing one Redis instance
// must not steal each other's reservations.
func New(client *goredis.Client, namespace string, ttl time.Duration) *Store {
	return &Store{client: client, namespace: namespace, ttl: ttl}
}

// key qualifies the application key.
func (s *Store) key(k domain.Key) string { return s.namespace + ":" + k.String() }

// Reserve implements ports.Reserve.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrIncomplete, req.Key)
	}

	raw, err := json.Marshal(envelope{Fingerprint: req.Fingerprint, Status: domain.StatusInFlight})
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("serialising the reservation: %w", err)
	}

	// SET NX sets the key AND its expiry in a single command: a crash between
	// the two would otherwise leave an everlasting reservation.
	won, err := s.client.SetNX(ctx, s.key(req.Key), raw, s.ttl).Result()
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("reserving the idempotency key: %w", err)
	}
	if won {
		return domain.Reservation{}, nil
	}
	return s.inspect(ctx, req)
}

// inspect reads the reservation that got ahead of us and settles the outcome.
func (s *Store) inspect(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	held, err := s.read(ctx, req.Key)
	if err != nil {
		// Key gone between the refused SET NX and this read, or failure: we
		// refuse. Making a client retry is benign, executing twice is not.
		if errors.Is(err, domain.ErrNotReserved) {
			return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrInFlight, req.Key)
		}
		return domain.Reservation{}, err
	}
	if held.Fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrConflict, req.Key)
	}
	if held.Status == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: held.Response}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrInFlight, req.Key)
}

// read loads the record. An absence returns domain.ErrNotReserved.
func (s *Store) read(ctx context.Context, key domain.Key) (envelope, error) {
	raw, err := s.client.Get(ctx, s.key(key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return envelope{}, fmt.Errorf("%w: key=%q", domain.ErrNotReserved, key)
	}
	if err != nil {
		return envelope{}, fmt.Errorf("reading the idempotency key: %w", err)
	}
	var held envelope
	if err := json.Unmarshal(raw, &held); err != nil {
		return envelope{}, fmt.Errorf("unreadable idempotency record: %w", err)
	}
	return held, nil
}

// Complete implements ports.Complete.
func (s *Store) Complete(ctx context.Context, key domain.Key, response []byte) error {
	held, err := s.read(ctx, key)
	if err != nil {
		return err
	}
	if held.Status != domain.StatusInFlight {
		return fmt.Errorf("%w: key=%q already resolved", domain.ErrNotReserved, key)
	}

	raw, err := json.Marshal(envelope{
		Fingerprint: held.Fingerprint,
		Status:      domain.StatusDone,
		Response:    response,
	})
	if err != nil {
		return fmt.Errorf("serialising the idempotent response: %w", err)
	}

	// KeepTTL: the replay window runs from the reservation. Without it, every
	// memorisation would push the expiry back and the keys would pile up.
	kept, err := s.client.SetXX(ctx, s.key(key), raw, goredis.KeepTTL).Result()
	if err != nil {
		return fmt.Errorf("memorising the idempotent response: %w", err)
	}
	if !kept {
		return fmt.Errorf("%w: key=%q expired during the operation", domain.ErrNotReserved, key)
	}
	return nil
}

// Release implements ports.Release.
//
// Only deletes a reservation in flight: erasing a completed key would reopen the
// door to the replay.
func (s *Store) Release(ctx context.Context, key domain.Key) error {
	held, err := s.read(ctx, key)
	if errors.Is(err, domain.ErrNotReserved) {
		return nil
	}
	if err != nil {
		return err
	}
	if held.Status != domain.StatusInFlight {
		return nil
	}
	if err := s.client.Del(ctx, s.key(key)).Err(); err != nil {
		return fmt.Errorf("releasing the idempotency key: %w", err)
	}
	return nil
}

// Purge implements ports.Purge.
//
// Without effect BY DESIGN, not by omission: Redis expires the keys itself.
// Returning 0 without error lets the scheduler call the port whichever driver is
// wired in.
func (s *Store) Purge(_ context.Context) (int64, error) { return 0, nil }
