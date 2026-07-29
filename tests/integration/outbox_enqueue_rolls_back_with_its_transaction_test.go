//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestOutboxEnqueueRollsBackWithItsTransaction verifies the very reason the
// transactional outbox exists (ADR 006).
//
// The promise states itself in one sentence and cannot be verified without a
// database: the event and the business data live or die TOGETHER. If either of
// the two could survive alone, the outbox would bring nothing but one more
// table.
//
// The two failure modes avoided are opposites, and both of them cost:
//
//   - the business write fails, the event goes out → a consumer acts on a fact
//     that never happened;
//   - the business write succeeds, the event is lost → nobody acts, and
//     nothing reports it.
//
// The second one is the worse: it produces no error at all.
//
// The test goes through `RunInTx`, the ONLY path that opens a transaction in
// this starter — there is deliberately no exported way to place one in a
// context. A `Result` in `Err` triggers the rollback: the business error is
// transactionally significant, and that is precisely what is exercised.
func TestOutboxEnqueueRollsBackWithItsTransaction(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	eventType := unique(t, "integration.rollback")
	unitOfWork := database.RunInTx[int, string](p)

	// A unit of work that writes and then FAILS.
	res := unitOfWork(ctx, func(txCtx context.Context) result.Result[int, string] {
		if _, err := store.Enqueue(txCtx, domain.NewMessage{
			Type:        eventType,
			AggregateID: "agg-cancelled",
			Payload:     []byte(`{}`),
		}); err != nil {
			t.Errorf("Enqueue inside the transaction: %v", err)
			return result.Err[int, string]("enqueue")
		}

		// The message MUST be visible here, otherwise an Enqueue that wrote
		// nothing at all would pass this test without anyone noticing.
		var inside int
		if err := database.QuerierFrom(txCtx, p).QueryRow(txCtx,
			"SELECT count(*) FROM platform.outbox_messages WHERE event_type = $1", eventType,
		).Scan(&inside); err != nil {
			t.Errorf("counting inside the transaction: %v", err)
			return result.Err[int, string]("count")
		}
		if inside != 1 {
			t.Errorf("%d message(s) visible INSIDE the transaction, want 1", inside)
		}

		// The business failure that must roll everything back.
		return result.Err[int, string]("the business write failed")
	})

	if res.IsOk() {
		t.Fatal("the unit of work was supposed to return an error")
	}

	// Seen from the POOL, outside any transaction: nothing left.
	var outside int
	if err := p.QueryRow(ctx,
		"SELECT count(*) FROM platform.outbox_messages WHERE event_type = $1", eventType,
	).Scan(&outside); err != nil {
		t.Fatalf("counting outside the transaction: %v", err)
	}
	if outside != 0 {
		t.Fatalf("%d message(s) survive the rollback: an event would go out for "+
			"a business write that never happened (ADR 006)", outside)
	}
}
