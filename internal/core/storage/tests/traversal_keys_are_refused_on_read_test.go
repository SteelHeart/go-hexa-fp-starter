package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestTraversalKeysAreRefusedOnRead: on READ, the key has not been derived by
// SafeKey — it comes from a URL, hence from a stranger.
//
// This is the hole that protection on write does not cover: hashing incoming
// names is useless if `GET /files/../../etc/passwd` is served. The check
// therefore takes place BEFORE touching the disk, for reading as well as for
// deletion.
func TestTraversalKeysAreRefusedOnRead(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	keys := map[string]domain.Key{
		"climb":               "../secret.pem",
		"deep climb":          "../../../etc/passwd",
		"absolute":            "/etc/passwd",
		"climb in the middle": "ab/../../cd",
		"bare parent":         "..",
		"empty":               "",
		"Windows":             "..\\secret.pem",
	}

	for name, key := range keys {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := mod.Get(ctx, key); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("Get(%q) = %v, want ErrUnsafeName", key, err)
			}
			if err := mod.Delete(ctx, key); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("Delete(%q) = %v, want ErrUnsafeName", key, err)
			}
		})
	}
}
