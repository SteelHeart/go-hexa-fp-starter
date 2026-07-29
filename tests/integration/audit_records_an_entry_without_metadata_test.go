//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	pgaudit "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/postgres"
)

// TestAuditRecordsAnEntryWithoutMetadata exercises the audit driver against the
// real table, without metadata.
//
// The question asked is the same one that brought the outbox driver down: what
// becomes of a nil map facing a `jsonb NOT NULL` column? Here the answer
// differs — `json.Marshal(nil)` produces the JSON literal `null`, which is a
// VALID jsonb value, so the constraint holds.
//
// It holds, but it stores `null` where the column announces `DEFAULT '{}'`. A
// consumer would then have to tell two shapes of "nothing" apart. This test
// pins the expected shape instead of leaving it to depend on an encoding
// detail.
//
// The timestamp comes from an injected port: that is what keeps the module
// testable without waiting, and what makes this check possible at all.
func TestAuditRecordsAnEntryWithoutMetadata(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	instant := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	record := pgaudit.New(p, func() time.Time { return instant })

	actor := unique(t, "integration-audit")
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.audit_log WHERE actor = $1", actor)
	})

	if err := record(ctx, domain.Entry{
		Actor:      actor,
		Action:     "integration.executed",
		EntityType: "run",
		EntityID:   "1",
		// Metadata deliberately absent.
	}); err != nil {
		t.Fatalf("an entry without metadata must be accepted: %v", err)
	}

	var (
		metadata   []byte
		occurredAt time.Time
	)
	if err := p.QueryRow(ctx,
		"SELECT metadata, occurred_at FROM platform.audit_log WHERE actor = $1", actor,
	).Scan(&metadata, &occurredAt); err != nil {
		t.Fatalf("reading the entry back: %v", err)
	}

	if got := string(metadata); got != "{}" {
		t.Errorf("stored metadata = %q, want \"{}\" — a consumer would otherwise "+
			"have to tell two shapes of \"no metadata\" apart", got)
	}
	if !occurredAt.UTC().Equal(instant) {
		t.Errorf("timestamp = %v, want %v: the instant must come from the injected "+
			"port, never from the database", occurredAt.UTC(), instant)
	}
}
