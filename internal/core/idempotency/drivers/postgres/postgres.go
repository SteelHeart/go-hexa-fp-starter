// Package postgres implémente l'idempotence sur PostgreSQL.
//
// # GARANTIES
//
//   - **Exclusivité entre répliques** : la contrainte d'unicité sur `key` tranche.
//     Deux requêtes concurrentes portant la même clé ne peuvent pas insérer
//     toutes les deux ; la perdante lit la ligne gagnante et refuse.
//   - **Durabilité** : au niveau de la base.
//
// # NON-GARANTIES
//
//   - **La réservation n'est PAS dans la transaction métier**, par conception :
//     elle passe par le pool, jamais par le querier du contexte. Une réservation
//     invisible aux autres connexions ne protégerait de rien — c'est l'inverse du
//     choix fait dans l'outbox, où l'atomicité avec le métier est justement le but.
//     Conséquence : après un échec métier, `Release` est obligatoire.
//   - **La fenêtre de rejeu court depuis la réservation**, pas depuis la fin de
//     l'opération : une opération longue raccourcit d'autant la mémorisation.
//   - **Aucune purge automatique.** `Purge` doit être appelée par l'ordonnanceur.
//
// # État
//
// Écrit, JAMAIS exécuté contre une base : la migration de `platform.idempotency_keys`
// n'existe pas encore (issue #2). Ne pas le présenter comme éprouvé.
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

// Store implémente l'idempotence sur PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
	ttl  time.Duration
}

// New construit le magasin.
func New(pool *pgxpool.Pool, ttl time.Duration) *Store {
	return &Store{pool: pool, ttl: ttl}
}

// reserveQuery réserve la clé, ou reprend une réservation expirée.
//
// `ON CONFLICT … DO UPDATE … WHERE expires_at < now()` est le cœur du mécanisme :
//   - clé libre → insertion, une ligne revient : nous tenons la réservation ;
//   - clé tenue par une réservation vivante → le WHERE bloque la mise à jour,
//     AUCUNE ligne ne revient : quelqu'un d'autre la tient ;
//   - clé tenue par une réservation expirée → reprise, une ligne revient.
//
// Un `DO UPDATE SET key = key` sans WHERE, lui, rendrait toujours une ligne : on
// ne saurait jamais si on vient de gagner la clé ou si on l'a volée à un appel en
// cours. C'est cette confusion qui laisse passer les doublons.
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

// Reserve implémente ports.Reserve.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrIncomplete, req.Key)
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
		return domain.Reservation{}, fmt.Errorf("réservation de la clé d'idempotence: %w", err)
	}
}

// inspect lit la réservation qui nous a devancés et tranche.
func (s *Store) inspect(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	var (
		fingerprint string
		status      string
		response    []byte
	)
	err := s.pool.QueryRow(ctx, inspectQuery, req.Key.String()).
		Scan(&fingerprint, &status, &response)

	if errors.Is(err, pgx.ErrNoRows) {
		// La ligne a disparu entre l'insertion refusée et cette lecture : un
		// Release concurrent. On refuse plutôt que de rejouer la réservation —
		// faire réessayer un client est bénin, exécuter deux fois ne l'est pas.
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrInFlight, req.Key)
	}
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("lecture de la clé d'idempotence: %w", err)
	}
	if fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrConflict, req.Key)
	}
	if domain.Status(status) == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: response}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrInFlight, req.Key)
}

// Complete implémente ports.Complete.
//
// Le WHERE porte sur le statut ET l'expiration : mémoriser une réponse sur une
// réservation morte ferait croire à une garantie qui n'a pas tenu.
func (s *Store) Complete(ctx context.Context, key domain.Key, response []byte) error {
	const query = `
		UPDATE platform.idempotency_keys
		SET status = $3, response = $2, completed_at = now()
		WHERE key = $1 AND status = $4 AND expires_at > now()`

	tag, err := s.pool.Exec(ctx, query, key.String(), response,
		string(domain.StatusDone), string(domain.StatusInFlight))
	if err != nil {
		return fmt.Errorf("mémorisation de la réponse idempotente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: clé=%q", domain.ErrNotReserved, key)
	}
	return nil
}

// Release implémente ports.Release.
func (s *Store) Release(ctx context.Context, key domain.Key) error {
	const query = `DELETE FROM platform.idempotency_keys WHERE key = $1 AND status = $2`
	if _, err := s.pool.Exec(ctx, query, key.String(), string(domain.StatusInFlight)); err != nil {
		return fmt.Errorf("libération de la clé d'idempotence: %w", err)
	}
	return nil
}

// Purge implémente ports.Purge.
func (s *Store) Purge(ctx context.Context) (int64, error) {
	const query = `DELETE FROM platform.idempotency_keys WHERE expires_at < now()`
	tag, err := s.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("purge des clés d'idempotence: %w", err)
	}
	return tag.RowsAffected(), nil
}
