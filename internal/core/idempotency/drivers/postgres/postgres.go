// Package postgres implements idempotency on PostgreSQL.
//
// # GUARANTEES
//
//   - **Exclusivity across replicas**: the uniqueness constraint on `key`
//     settles it. Two concurrent requests carrying the same key cannot both
//     insert; the loser reads the winning row and refuses.
//   - **Durability**: at the database level.
//
// # NON-GUARANTEES
//
//   - **The reservation is NOT in the business transaction**, by design: it goes
//     through the pool, never through the querier of the context. A reservation
//     invisible to the other connections would protect nothing — this is the
//     opposite of the choice made in the outbox, where atomicity with the
//     business work is precisely the point.
//     Consequence: after a business failure, `Release` is mandatory.
//   - **The replay window runs from the reservation**, not from the end of the
//     operation: a long operation shortens the memorisation by as much.
//   - **No automatic purge.** `Purge` must be called by the scheduler.
//
// # Status
//
// EXERCISED against a real Postgres since #37: `tests/integration` covers the
// exclusivity of a reservation under concurrency, the refusal of a reused key
// with another payload, and expiry. The CI job of the same name runs it on
// every pull request.
//
// ⚠️ This block used to say "NEVER run against a database, the migration does
// not exist yet (issue #2)". Both halves stopped being true with #5, #84 and
// #37 — and the sentence asking not to present the driver as exercised was the
// one presenting it as untested.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// Store implements idempotency on PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
	ttl  time.Duration
}

// New builds the store.
func New(pool *pgxpool.Pool, ttl time.Duration) *Store {
	return &Store{pool: pool, ttl: ttl}
}

// reserveQuery reserves the key, or takes over an expired reservation.
//
// `ON CONFLICT … DO UPDATE … WHERE expires_at < now()` is the heart of the
// mechanism:
//   - free key → insertion, one row comes back: we hold the reservation;
//   - key held by a live reservation → the WHERE blocks the update, NO row comes
//     back: someone else holds it;
//   - key held by an expired reservation → take-over, one row comes back.
//
// A `DO UPDATE SET key = key` without a WHERE, on the other hand, would always
// return a row: one would never know whether the key has just been won or stolen
// from a call in flight. It is that confusion which lets duplicates through.
const reserveQuery = `
	INSERT INTO platform.idempotency_keys (key, fingerprint, status, expires_at)
	VALUES ($1, $2, $3, now() + make_interval(secs => $4))
	ON CONFLICT (key) DO UPDATE
		SET fingerprint = excluded.fingerprint,
		    status      = excluded.status,
		    response    = NULL,
		    expires_at  = excluded.expires_at
		WHERE idempotency_keys.expires_at < now()
	RETURNING key`

const inspectQuery = `
	SELECT fingerprint, status, response
	FROM platform.idempotency_keys
	WHERE key = $1`

// Reserve implements ports.Reserve.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrIncomplete, req.Key)
	}

	var granted string
	err := s.pool.QueryRow(ctx, reserveQuery,
		req.Key.String(), req.Fingerprint, string(domain.StatusInFlight), s.ttl.Seconds(),
	).Scan(&granted)

	switch {
	case err == nil:
		return domain.Reservation{}, nil
	case errors.Is(err, pgx.ErrNoRows):
		return s.inspect(ctx, req)
	default:
		return domain.Reservation{}, fmt.Errorf("reserving the idempotency key: %w", err)
	}
}

// inspect reads the reservation that got ahead of us and settles the outcome.
func (s *Store) inspect(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	var (
		fingerprint string
		status      string
		response    []byte
	)
	err := s.pool.QueryRow(ctx, inspectQuery, req.Key.String()).
		Scan(&fingerprint, &status, &response)

	if errors.Is(err, pgx.ErrNoRows) {
		// The row disappeared between the refused insertion and this read: a
		// concurrent Release. We refuse rather than replay the reservation —
		// making a client retry is benign, executing twice is not.
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrInFlight, req.Key)
	}
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("reading the idempotency key: %w", err)
	}
	if fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrConflict, req.Key)
	}
	if domain.Status(status) == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: response}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: key=%q", domain.ErrInFlight, req.Key)
}

// Complete implements ports.Complete.
//
// The WHERE covers the status AND the expiry: memorising a response on a dead
// reservation would suggest a guarantee that did not hold.
func (s *Store) Complete(ctx context.Context, key domain.Key, response []byte) error {
	const query = `
		UPDATE platform.idempotency_keys
		SET status = $3, response = $2, completed_at = now()
		WHERE key = $1 AND status = $4 AND expires_at > now()`

	tag, err := s.pool.Exec(ctx, query, key.String(), response,
		string(domain.StatusDone), string(domain.StatusInFlight))
	if err != nil {
		return fmt.Errorf("memorising the idempotent response: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: key=%q", domain.ErrNotReserved, key)
	}
	return nil
}

// Release implements ports.Release.
func (s *Store) Release(ctx context.Context, key domain.Key) error {
	const query = `DELETE FROM platform.idempotency_keys WHERE key = $1 AND status = $2`
	if _, err := s.pool.Exec(ctx, query, key.String(), string(domain.StatusInFlight)); err != nil {
		return fmt.Errorf("releasing the idempotency key: %w", err)
	}
	return nil
}

// Purge implements ports.Purge.
func (s *Store) Purge(ctx context.Context) (int64, error) {
	const query = `DELETE FROM platform.idempotency_keys WHERE expires_at < now()`
	tag, err := s.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("purging the idempotency keys: %w", err)
	}
	return tag.RowsAffected(), nil
}
