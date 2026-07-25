// Package log implémente l'audit vers le journal applicatif.
//
// # NON-GARANTIES
//
//   - **Ni requêtable** : retrouver « toutes les actions de cet acteur » exige
//     de fouiller des logs.
//   - **Ni inaltérable** : un log se tronque, se réécrit et s'efface.
//   - **Ni conservé** : la rétention est celle de l'agent de journaux.
//
// Convient en développement. Ne convient PAS dès qu'une preuve est attendue —
// argent, consentement, accès à une donnée sensible.
package log

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/ports"
)

// New construit le pilote.
func New(logger *slog.Logger, now func() time.Time) ports.Record {
	return func(ctx context.Context, entry domain.Entry) error {
		if !entry.IsComplete() {
			return fmt.Errorf("%w: action=%q entité=%q", domain.ErrIncomplete, entry.Action, entry.EntityType)
		}
		stamped := entry.WithTime(now())
		logger.InfoContext(ctx, "audit",
			slog.String("actor", stamped.Actor),
			slog.String("action", stamped.Action),
			slog.String("entity_type", stamped.EntityType),
			slog.String("entity_id", stamped.EntityID),
			slog.Time("occurred_at", stamped.At),
			slog.Any("metadata", stamped.Metadata),
		)
		return nil
	}
}
