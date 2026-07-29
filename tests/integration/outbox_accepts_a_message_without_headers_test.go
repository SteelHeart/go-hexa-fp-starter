//go:build integration

package integration

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
)

// TestOutboxAcceptsAMessageWithoutHeaders is the witness of a REAL defect,
// found by this level of test on its very first run.
//
// The driver passed `msg.Headers` explicitly to an `INSERT`. A nil map arrives
// as NULL, the column is `NOT NULL`, and the migration's `DEFAULT '{}'::jsonb`
// does not apply — a DEFAULT only kicks in when the column is OMITTED.
//
// Exact reach of the defect: `outboxpub`, the real adapter of the reference
// module, sets NO header at all. So **`POST /v1/users` would have failed in
// production as soon as `modules.outbox.driver` is `postgres`.**
//
// Why 285 unit tests did not see it: they all run on the `memory` driver,
// which accepts nil without flinching. The two drivers did not honour the same
// contract, and nothing could say so — which is very exactly what the
// `integration` level exists to catch (#37).
//
// ⚠️ Do not "simplify" this test on the grounds that it looks like an ordinary
// Enqueue. It is the absence of headers that is exercised, and that alone.
func TestOutboxAcceptsAMessageWithoutHeaders(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	eventType := unique(t, "integration.no-headers")
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t),
			"DELETE FROM platform.outbox_messages WHERE event_type = $1", eventType)
	})

	id, err := store.Enqueue(ctx, domain.NewMessage{
		Type:        eventType,
		AggregateID: "agg-no-headers",
		Payload:     []byte(`{"k":"v"}`),
		// Headers and TraceParent deliberately absent: this is the shape
		// `outboxpub` produces, hence the shape of the real path.
	})
	if err != nil {
		t.Fatalf("a message without headers must be accepted, as on the memory driver: %v", err)
	}
	if id == "" {
		t.Fatal("empty identifier")
	}

	// The stored header must be an EMPTY map, never NULL: a consumer reading
	// `headers` must not have to tell two shapes of "nothing" apart.
	var headers map[string]string
	if err := p.QueryRow(ctx,
		"SELECT headers FROM platform.outbox_messages WHERE id = $1", string(id),
	).Scan(&headers); err != nil {
		t.Fatalf("reading the headers back: %v", err)
	}
	if headers == nil {
		t.Error("headers read back as NULL: normalisation did not happen")
	}
	if len(headers) != 0 {
		t.Errorf("headers read back = %v, want an empty map", headers)
	}
}
