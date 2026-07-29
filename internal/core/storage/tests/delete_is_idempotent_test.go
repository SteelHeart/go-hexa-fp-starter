package tests

import (
	"context"
	"testing"
)

// TestDeleteIsIdempotent: deleting an absent object is not an error.
//
// The caller wants the object to be gone, and it is gone. Making an idempotent
// deletion fail would force every caller to distinguish two equivalent cases —
// and the first one to forget the distinction would propagate an error for a
// success.
func TestDeleteIsIdempotent(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	located, err := mod.Put(ctx, object("temporary.txt", "content"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := mod.Delete(ctx, located.Key); err != nil {
		t.Fatalf("first deletion: %v", err)
	}
	if err := mod.Delete(ctx, located.Key); err != nil {
		t.Errorf("second deletion = %v, want nil", err)
	}

	if _, err := mod.Get(ctx, located.Key); err == nil {
		t.Error("the deleted object must no longer be readable")
	}
}
