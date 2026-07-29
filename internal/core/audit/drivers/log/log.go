// Package log implements auditing towards the application log.
//
// # NON-GUARANTEES
//
//   - **Not queryable**: finding "every action of this actor" requires digging
//     through logs.
//   - **Not tamper-proof**: a log gets truncated, rewritten and erased.
//   - **Not retained**: retention is that of the log agent.
//
// Suitable in development. NOT suitable as soon as proof is expected — money,
// consent, access to sensitive data.
package log

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/ports"
)

// New builds the driver.
func New(logger *slog.Logger, now func() time.Time) ports.Record {
	return func(ctx context.Context, entry domain.Entry) error {
		if !entry.IsComplete() {
			return fmt.Errorf("%w: action=%q entity=%q", domain.ErrIncomplete, entry.Action, entry.EntityType)
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
