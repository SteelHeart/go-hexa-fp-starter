package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestURLIsBuiltFromBaseURL: the returned URL prefixes the key with the
// configured base address, without ever doubling the slash.
//
// The caller stores this URL in the database. One slash too many makes it valid
// one day and broken the next depending on the server that serves it, and URLs
// already persisted are not fixed by a deployment.
func TestURLIsBuiltFromBaseURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"without trailing slash": "/files",
		"with trailing slash":    "/files/",
		"absolute address":       "https://cdn.example.test/files/",
	}

	for name, base := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mod, err := storage.New(config.Module{
				Enabled: true,
				Driver:  "disk",
				Options: map[string]any{"base_dir": t.TempDir(), "base_url": base},
			}, storage.Deps{})
			if err != nil {
				t.Fatalf("construction: %v", err)
			}

			located, err := mod.Put(context.Background(), object("doc.pdf", "x"))
			if err != nil {
				t.Fatalf("Put: %v", err)
			}

			want := strings.TrimSuffix(base, "/") + "/" + located.Key.String()
			if located.URL != want {
				t.Errorf("URL = %q, want %q", located.URL, want)
			}
			if strings.Contains(strings.TrimPrefix(located.URL, "https://"), "//") {
				t.Errorf("URL = %q contains a doubled slash", located.URL)
			}
		})
	}
}
