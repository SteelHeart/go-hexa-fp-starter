// Package postgres implémente l'audit dans une table en ajout seul.
//
// # GARANTIES
//
//   - **Atomicité avec la transaction métier** : l'écriture passe par le querier
//     du contexte. Un fait annulé ne laisse pas de trace mensongère.
//   - **Inaltérabilité** : la migration RÉVOQUE `UPDATE` et `DELETE` sur la table
//     pour le rôle applicatif. Ce n'est pas une intention, c'est une contrainte —
//     un journal qu'on peut réécrire ne prouve rien.
//
// # NON-GARANTIES
//
//   - La table croît sans fin. Sa rétention est une décision d'exploitation,
//     jamais une suppression automatique.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// New construit le pilote.
func New(pool *pgxpool.Pool, now func() time.Time) ports.Record {
	return func(ctx context.Context, entry domain.Entry) error {
		if !entry.IsComplete() {
			return fmt.Errorf("%w: action=%q entité=%q", domain.ErrIncomplete, entry.Action, entry.EntityType)
		}
		// Une carte nil se sérialise en littéral JSON `null`, pas en `{}`.
		//
		// Ça ne CASSE pas : `null` est un jsonb valide, donc la contrainte
		// NOT NULL passe. C'est précisément ce qui rend ce défaut durable — il
		// ne se signale jamais, contrairement à son jumeau du pilote d'outbox,
		// qui lui refusait l'insertion (#37).
		//
		// L'effet réel est chez le lecteur : le journal d'audit est conservé
		// longtemps et relu pendant un incident. Y trouver deux formes de
		// « aucune métadonnée » — `null` et `{}` — selon la version du code qui
		// a écrit la ligne est exactement ce qu'on ne veut pas au pire moment.
		champs := entry.Metadata
		if champs == nil {
			champs = map[string]any{}
		}
		metadata, err := json.Marshal(champs)
		if err != nil {
			return fmt.Errorf("sérialisation des métadonnées d'audit: %w", err)
		}

		const query = `
			INSERT INTO platform.audit_log
				(actor, action, entity_type, entity_id, metadata, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6)`

		stamped := entry.WithTime(now())
		if _, err := database.QuerierFrom(ctx, pool).Exec(ctx, query,
			stamped.Actor, stamped.Action, stamped.EntityType, stamped.EntityID,
			metadata, stamped.At,
		); err != nil {
			return fmt.Errorf("écriture du journal d'audit: %w", err)
		}
		return nil
	}
}
