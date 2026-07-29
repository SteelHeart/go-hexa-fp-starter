package tests

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestObjectWithoutContentIsRefused: an object without a name or without a
// stream is refused before any write.
//
// A nil `Content` would make io.Copy panic in the driver: the explicit refusal
// turns a process crash into an error the caller can handle.
func TestObjectWithoutContentIsRefused(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	cases := map[string]domain.Object{
		"without stream": {Name: "doc.pdf", Content: nil},
		"without name":   {Name: "", Content: strings.NewReader("x")},
		"empty":          {},
	}

	for name, obj := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := mod.Put(ctx, obj); !errors.Is(err, domain.ErrEmptyContent) {
				t.Errorf("Put = %v, want ErrEmptyContent", err)
			}
		})
	}
}
