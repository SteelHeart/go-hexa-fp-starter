//go:build integration

package integration

import (
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
)

// TestOutboxClaimNeverHandsTheSameMessageTwice exercises the heart of the
// mechanism: `FOR UPDATE SKIP LOCKED`.
//
// This is THE property no unit test can reach. The `memory` driver simulates
// it; here, two real Postgres transactions compete for the same rows.
//
// What would break without it: two replicas of the dispatcher would publish
// the same event. The consumer would see a duplicate — and a duplicated
// business event is an email sent twice, or worse, a payment replayed.
//
// The test claims from TWO concurrent transactions, each held open while the
// other claims. Without `SKIP LOCKED`, the second would wait for the lock and
// then return the same rows; with it, it skips them.
func TestOutboxClaimNeverHandsTheSameMessageTwice(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	// An event type specific to this test: the table is shared, and another
	// test running in parallel must not skew the count.
	eventType := unique(t, "integration.claim")

	const total = 6
	for range total {
		if _, err := store.Enqueue(ctx, domain.NewMessage{
			Type:        eventType,
			AggregateID: "agg-1",
			Payload:     []byte(`{"k":"v"}`),
		}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), //nolint:errcheck // cleanup, its failure says nothing useful
			"DELETE FROM platform.outbox_messages WHERE event_type = $1", eventType)
	})

	// Two transactions, open at the same time. Each one claims, then waits for
	// the other to have claimed before committing: it is that simultaneity
	// which puts `SKIP LOCKED` to the test.
	var wg sync.WaitGroup
	seen := make([][]domain.MessageID, 2)
	var mu sync.Mutex
	ready := make(chan struct{})

	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			tx, err := p.Begin(ctx)
			if err != nil {
				t.Errorf("Begin: %v", err)
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()

			rows, err := tx.Query(ctx, `
				SELECT id FROM platform.outbox_messages
				WHERE status = 'pending' AND event_type = $1
				ORDER BY available_at LIMIT 3
				FOR UPDATE SKIP LOCKED`, eventType)
			if err != nil {
				t.Errorf("Query: %v", err)
				return
			}
			var ids []domain.MessageID
			for rows.Next() {
				var id domain.MessageID
				if err := rows.Scan(&id); err != nil {
					t.Errorf("Scan: %v", err)
					rows.Close()
					return
				}
				ids = append(ids, id)
			}
			rows.Close()

			mu.Lock()
			seen[i] = ids
			mu.Unlock()

			// Both transactions stay open until here: without this barrier,
			// the first could release its locks before the second claims, and
			// the test would pass without proving anything.
			<-ready
		}()
	}

	// Let both claim, then release them.
	for {
		mu.Lock()
		done := len(seen[0]) > 0 && len(seen[1]) > 0
		mu.Unlock()
		if done {
			break
		}
	}
	close(ready)
	wg.Wait()

	if len(seen[0]) == 0 || len(seen[1]) == 0 {
		t.Fatalf("one transaction claimed nothing: %d and %d — the test would prove nothing",
			len(seen[0]), len(seen[1]))
	}

	crossed := map[domain.MessageID]bool{}
	for _, id := range seen[0] {
		crossed[id] = true
	}
	for _, id := range seen[1] {
		if crossed[id] {
			t.Fatalf("message %s was claimed by BOTH transactions: "+
				"FOR UPDATE SKIP LOCKED no longer protects, a replicated dispatcher would publish duplicates", id)
		}
	}
	if got := len(seen[0]) + len(seen[1]); got != total {
		t.Errorf("%d messages claimed in total, want %d — rows were lost or duplicated",
			got, total)
	}
}
