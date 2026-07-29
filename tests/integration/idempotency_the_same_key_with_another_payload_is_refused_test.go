//go:build integration

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	pgidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
)

// TestIdempotencyTheSameKeyWithAnotherPayloadIsRefused: one key covers only ONE
// request.
//
// The failure mode avoided is quiet and serious. A client reuses its
// idempotency key — by copying it, by faulty generation, or maliciously — with
// a different payload. Without this check, the starter would return the
// memorised response of the FIRST request for the SECOND. The client would
// believe its second transfer carried out; it would never have been.
//
// The refusal must therefore be explicit, and distinct from "already in
// flight".
func TestIdempotencyTheSameKeyWithAnotherPayloadIsRefused(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgidem.New(p, time.Hour)

	key := domain.Key(unique(t, "integration-conflict"))
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.idempotency_keys WHERE key = $1", key.String())
	})

	first := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]int{"amount": 100})}
	if _, err := store.Reserve(ctx, first); err != nil {
		t.Fatalf("first reservation: %v", err)
	}
	if err := store.Complete(ctx, key, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Identical replay: the memorised response comes back, and the operation
	// must NOT be run again.
	replay, err := store.Reserve(ctx, first)
	if err != nil {
		t.Fatalf("identical replay: %v", err)
	}
	if !replay.Replayed {
		t.Fatal("an identical replay must be reported as a replay, otherwise the operation runs again")
	}
	if string(replay.Response) != `{"ok":true}` {
		t.Errorf("memorised response = %q", replay.Response)
	}

	// Same key, OTHER payload: refusal.
	second := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]int{"amount": 999})}
	if _, err := store.Reserve(ctx, second); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict — the response of ANOTHER request "+
			"would be returned to the client, who would believe its operation carried out", err)
	}
}
