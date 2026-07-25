// Package postgres implémente l'outbox sur PostgreSQL.
//
// # GARANTIES
//
//   - **Atomicité avec la transaction métier** : `Enqueue` écrit via le querier
//     du contexte, donc dans la transaction en cours. Un rollback emporte
//     l'événement. C'est la raison d'être de ce patron.
//   - **Exclusivité de Claim entre répliques** : `FOR UPDATE SKIP LOCKED`.
//     Plusieurs workers dépilent sans coordination et sans doublon.
//   - **Durabilité** : au niveau de la base.
//
// # NON-GARANTIES
//
//   - Livraison « au moins une fois », jamais « exactement une fois ». Tout
//     consommateur doit être idempotent.
//   - Un message `failed` n'est jamais supprimé, donc la table croît : sa purge
//     est une décision d'exploitation, pas un comportement automatique.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Store implémente l'outbox sur PostgreSQL.
type Store struct{ pool *pgxpool.Pool }

// New construit le magasin.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Enqueue implémente ports.Enqueue.
//
// Passe par database.QuerierFrom : la même requête s'exécute dans la
// transaction courante s'il y en a une, sur le pool sinon. C'est ce qui rend
// l'atomicité transparente pour l'appelant.
func (s *Store) Enqueue(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error) {
	const query = `
		INSERT INTO platform.outbox_messages
			(id, event_type, aggregate_id, payload, trace_parent, headers)
		VALUES ($1, $2, $3, $4, $5, $6)`

	id := uuid.NewString()
	_, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query,
		id, msg.Type, msg.AggregateID, msg.Payload, msg.TraceParent, msg.Headers,
	)
	if err != nil {
		return "", fmt.Errorf("insertion dans l'outbox (%s): %w", msg.Type, err)
	}
	return domain.MessageID(id), nil
}

// claimQuery réserve un lot de messages dus.
//
// `FOR UPDATE SKIP LOCKED` est le cœur du mécanisme : chaque réplique saute les
// lignes que les autres tiennent déjà, donc aucune coordination externe n'est
// nécessaire et aucun message n'est traité deux fois en parallèle.
const claimQuery = `
	SELECT id, event_type, aggregate_id, payload, trace_parent, headers,
	       status, attempts, created_at, available_at
	FROM platform.outbox_messages
	WHERE status = 'pending' AND available_at <= now()
	ORDER BY available_at
	LIMIT $1
	FOR UPDATE SKIP LOCKED`

// Claim implémente ports.Claim.
func (s *Store) Claim(ctx context.Context, limit int) ([]domain.Message, error) {
	rows, err := database.QuerierFrom(ctx, s.pool).Query(ctx, claimQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("réservation dans l'outbox: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0, limit)
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(
			&m.ID, &m.Type, &m.AggregateID, &m.Payload, &m.TraceParent, &m.Headers,
			&m.Status, &m.Attempts, &m.CreatedAt, &m.AvailableAt,
		); err != nil {
			return nil, fmt.Errorf("lecture d'un message de l'outbox: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours de l'outbox: %w", err)
	}
	return messages, nil
}

// MarkDone implémente ports.MarkDone.
func (s *Store) MarkDone(ctx context.Context, id domain.MessageID) error {
	const query = `
		UPDATE platform.outbox_messages
		SET status = 'done', processed_at = now()
		WHERE id = $1`
	if _, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query, id.String()); err != nil {
		return fmt.Errorf("marquage du message %s comme traité: %w", id, err)
	}
	return nil
}

// MarkFailed implémente ports.MarkFailed.
//
// N'applique aucune politique : le calcul du recul et la décision d'abandon
// viennent de domain.NextAttempt, qui est pur et testé.
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
		return fmt.Errorf("enregistrement de l'échec du message %s: %w", attempt.ID, err)
	}
	return nil
}

// PendingCount implémente ports.PendingCount.
func (s *Store) PendingCount(ctx context.Context) (int64, error) {
	const query = `SELECT count(*) FROM platform.outbox_messages WHERE status = 'pending'`
	var count int64
	if err := database.QuerierFrom(ctx, s.pool).QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("comptage de l'outbox: %w", err)
	}
	return count, nil
}
