package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestClaimIsExclusive checks the most important contract of the port: two
// concurrent claims NEVER return the same message. The postgres driver obtains
// this through FOR UPDATE SKIP LOCKED; the memory driver must guarantee it too,
// otherwise it would lie about production behaviour.
func TestClaimIsExclusive(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	if _, err := mod.Enqueue(ctx, domain.NewMessage{Type: "t"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	first, err := mod.Claim(ctx, 10)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim: %d message(s), err=%v", len(first), err)
	}
	second, err := mod.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("an already claimed message was returned a second time: %d", len(second))
	}
}
