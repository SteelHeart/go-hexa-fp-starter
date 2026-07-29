// Package postgres implements auditing in an append-only table.
//
// # GUARANTEES
//
//   - **Atomicity with the business transaction**: the write goes through the
//     querier of the context. A rolled back fact leaves no lying trace.
//   - **Tamper-resistance**: the migration REVOKES `UPDATE` and `DELETE` on the
//     table for the application role. This is not an intention, it is a
//     constraint — a log one can rewrite proves nothing.
//
// # NON-GUARANTEES
//
//   - The table grows without end. Its retention is an operations decision,
//     never an automatic deletion.
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

// New builds the driver.
func New(pool *pgxpool.Pool, now func() time.Time) ports.Record {
	return func(ctx context.Context, entry domain.Entry) error {
		if !entry.IsComplete() {
			return fmt.Errorf("%w: action=%q entity=%q", domain.ErrIncomplete, entry.Action, entry.EntityType)
		}
		// A nil map serialises to the JSON literal `null`, not to `{}`.
		//
		// This does not BREAK anything: `null` is valid jsonb, so the NOT NULL
		// constraint passes. That is precisely what makes this defect durable —
		// it never signals itself, unlike its twin in the outbox driver, which
		// refused the insert (#37).
		//
		// The real effect is on the reader: the audit log is kept for a long
		// time and re-read during an incident. Finding two forms of "no
		// metadata" there — `null` and `{}` — depending on the version of the
		// code that wrote the row is exactly what one does not want at the
		// worst moment.
		fields := entry.Metadata
		if fields == nil {
			fields = map[string]any{}
		}
		metadata, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("serialising the audit metadata: %w", err)
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
			return fmt.Errorf("writing the audit log: %w", err)
		}
		return nil
	}
}
