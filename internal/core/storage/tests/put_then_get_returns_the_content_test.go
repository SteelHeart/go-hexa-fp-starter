package tests

import (
	"context"
	"io"
	"testing"
)

// TestPutThenGetReturnsTheContent: the nominal path, without which none of the
// security guarantees would be of any interest.
func TestPutThenGetReturnsTheContent(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()
	const content = "content of the receipt"

	located, err := mod.Put(ctx, object("receipt.pdf", content))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if located.Key == "" {
		t.Fatal("Put must return a key")
	}

	reader, err := mod.Get(ctx, located.Key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = reader.Close() }()

	read, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(read) != content {
		t.Errorf("content read back = %q, want %q", read, content)
	}
}
