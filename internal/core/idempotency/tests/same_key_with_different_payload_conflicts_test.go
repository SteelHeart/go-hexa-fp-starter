package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestSameKeyWithDifferentPayloadConflicts: the same key must designate the same
// request. We refuse rather than guess which of the two is the right one.
func TestSameKeyWithDifferentPayloadConflicts(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", map[string]int{"amount": 100})); err != nil {
		t.Fatalf("first reservation refused: %v", err)
	}
	_, err := mod.Reserve(ctx, request("k1", map[string]int{"amount": 999}))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("error = %v, want ErrConflict", err)
	}
}
