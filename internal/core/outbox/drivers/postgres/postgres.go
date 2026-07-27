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

	// UUID v7 et non v4 : la clé primaire est ORDONNÉE DANS LE TEMPS.
	//
	// Ce n'est pas un détail sur cette table. Un v4 aléatoire disperse les
	// insertions sur tout l'index, ce qui multiplie les pages sales et fragmente
	// le B-tree ; l'outbox étant la table la plus écrite du socle, elle est
	// exactement l'endroit où ça coûte le plus. Le v7 insère en queue d'index,
	// et il rend en prime `ORDER BY id` équivalent à l'ordre de création.
	// Imposé par rules/donnees-et-migrations.md §7.
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("génération de l'identifiant du message (%s): %w", msg.Type, err)
	}

	// Une carte nil arrive en NULL, et la colonne est NOT NULL.
	//
	// Le `DEFAULT '{}'::jsonb` de la migration ne sauve pas : un DEFAULT ne
	// s'applique que si la colonne est OMISE de l'INSERT, or elle est ici
	// passée explicitement. Sans cette normalisation, tout message sans
	// en-têtes est REFUSÉ par ce pilote — et `outboxpub`, l'adaptateur réel du
	// module de référence, n'en pose aucun. Autrement dit : `POST /v1/users`
	// aurait échoué en production sur `driver: postgres`.
	//
	// Le pilote `memory` accepte nil sans broncher. Les deux pilotes ne
	// respectaient donc PAS le même contrat, et les 285 tests unitaires ne
	// pouvaient pas le voir : ils tournent tous sur `memory`. C'est ce défaut
	// exact qui justifie le niveau `integration` (#37).
	//
	// nil signifie « aucun en-tête », pas « en-têtes inconnus » : la carte vide
	// est la traduction fidèle.
	headers := msg.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	if _, err := database.QuerierFrom(ctx, s.pool).Exec(ctx, query,
		id.String(), msg.Type, msg.AggregateID, msg.Payload, msg.TraceParent, headers,
	); err != nil {
		return "", fmt.Errorf("insertion dans l'outbox (%s): %w", msg.Type, err)
	}
	return domain.MessageID(id.String()), nil
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
