// Package postgres implements the outbox on PostgreSQL.
//
// # GUARANTEES
//
//   - **Atomicity with the business transaction**: `Enqueue` writes through the
//     querier of the context, hence within the current transaction. A rollback
//     takes the event with it. This is the whole point of this pattern.
//   - **Exclusivity of Claim between replicas**: `FOR UPDATE SKIP LOCKED`.
//     Several workers dispatch without coordination and without duplicates.
//   - **Durability**: at the database level.
//
// # NON-GUARANTEES
//
//   - « At least once » delivery, never « exactly once ». Every consumer must
//     be idempotent.
//   - A `failed` message is never deleted, so the table grows: purging it is an
//     operations decision, not an automatic behaviour.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Store implements the outbox on PostgreSQL.
type Store struct{ pool *pgxpool.Pool }

// New builds the store.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Enqueue implements ports.Enqueue.
//
// Goes through database.QuerierFrom: the same query runs inside the current
// transaction if there is one, on the pool otherwise. That is what makes
// atomicity transparent for the caller.
func (s *Store) Enqueue(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error) {
	const query = `
		INSERT INTO platform.outbox_messages
			(id, event_type, aggregate_id, payload, trace_parent, headers)
		VALUES ($1, $2, $3, $4, $5, $6)`

	// UUID v7 and not v4: the primary key is ORDERED IN TIME.
	//
	// This is not a detail on this table. A random v4 scatters the inserts over
	// the whole index, which multiplies dirty pages and fragments the B-tree;
	// the outbox being the most written table of the starter, it is exactly the
	// place where this costs the most. The v7 inserts at the tail of the index,
	// and it makes `ORDER BY id` equivalent to creation order as a bonus.
	// Imposed by rules/donnees-et-migrations.md §7.
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generating the message identifier (%s): %w", msg.Type, err)
	}

	// A nil map arrives as NULL, and the column is NOT NULL.
	//
	// The `DEFAULT '{}'::jsonb` of the migration does not save it: a DEFAULT
	// only applies if the column is OMITTED from the INSERT, whereas here it is
	// passed explicitly. Without this normalisation, every message without
	// headers is REFUSED by this driver — and `outboxpub`, the real adapter of
	// the reference module, sets none. In other words: `POST /v1/users` would
	// have failed in production on `driver: postgres`.
	//
	// The `memory` driver accepts nil without flinching. The two drivers
	// therefore did NOT honour the same contract, and the 285 unit tests could
	// not see it: they all run on `memory`. It is this exact defect that
	// justifies the `integration` level (#37).
	//
	// nil means « no headers », not « unknown headers »: the empty map is the
	// faithful translation.
	headers := msg.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	if _, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query,
		id.String(), msg.Type, msg.AggregateID, msg.Payload, msg.TraceParent, headers,
	); err != nil {
		return "", fmt.Errorf("inserting into the outbox (%s): %w", msg.Type, err)
	}
	return domain.MessageID(id.String()), nil
}

// claimQuery claims a batch of due messages.
//
// `FOR UPDATE SKIP LOCKED` is the heart of the mechanism: each replica skips
// the rows the others already hold, so no external coordination is necessary
// and no message is processed twice in parallel.
const claimQuery = `
	SELECT id, event_type, aggregate_id, payload, trace_parent, headers,
	       status, attempts, created_at, available_at
	FROM platform.outbox_messages
	WHERE status = 'pending' AND available_at <= now()
	ORDER BY available_at
	LIMIT $1
	FOR UPDATE SKIP LOCKED`

// Claim implements ports.Claim.
func (s *Store) Claim(ctx context.Context, limit int) ([]domain.Message, error) {
	rows, err := database.QuerierFrom(ctx, s.pool).Query(ctx, claimQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("claiming in the outbox: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0, limit)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(
			&m.ID, &m.Type, &m.AggregateID, &m.Payload, &m.TraceParent, &m.Headers,
			&m.Status, &m.Attempts, &m.CreatedAt, &m.AvailableAt,
		); err != nil {
			return nil, fmt.Errorf("reading an outbox message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("walking the outbox: %w", err)
	}
	return messages, nil
}

// MarkDone implements ports.MarkDone.
func (s *Store) MarkDone(ctx context.Context, id domain.MessageID) error {
	const query = `
		UPDATE platform.outbox_messages
		SET status = 'done', processed_at = now()
		WHERE id = $1`
	if _, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query, id.String()); err != nil {
		return fmt.Errorf("marking message %s as processed: %w", id, err)
	}
	return nil
}

// MarkFailed implements ports.MarkFailed.
//
// Applies no policy: the backoff computation and the decision to give up come
// from domain.NextAttempt, which is pure and tested.
func (s *Store) MarkFailed(ctx context.Context, attempt domain.FailedAttempt) error {
	const query = `
		UPDATE platform.outbox_messages
		SET attempts = $2, status = $3, available_at = $4, last_error = $5
		WHERE id = $1`
	_, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query,
		attempt.ID.String(), attempt.Attempts, string(attempt.Status),
		attempt.AvailableAt, attempt.Reason,
	)
	if err != nil {
		return fmt.Errorf("recording the failure of message %s: %w", attempt.ID, err)
	}
	return nil
}

// PendingCount implements ports.PendingCount.
func (s *Store) PendingCount(ctx context.Context) (int64, error) {
	const query = `SELECT count(*) FROM platform.outbox_messages WHERE status = 'pending'`
	var count int64
	if err := database.QuerierFrom(ctx, s.pool).QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting the outbox: %w", err)
	}
	return count, nil
}
