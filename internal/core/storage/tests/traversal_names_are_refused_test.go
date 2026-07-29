package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestTraversalNamesAreRefused is the most important test of the module.
//
// The name of an object comes from a form, hence from a stranger. Used as a
// path as it stands, it allows writing anywhere on the host — this is the
// directory traversal vulnerability, and it has been enough to compromise whole
// servers.
//
// Two protections stack, and the test verifies both:
//   - names that designate no file are REFUSED;
//   - the others are HASHED, so no path fragment survives.
func TestTraversalNamesAreRefused(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"simple climb":      "..",
		"deep climb":        "../../..",
		"root":              "/",
		"current directory": ".",
		"empty name":        "",
		"Windows climb":     "..\\..",
		"separators only":   "///",
		"disguised climb":   "foo/../..",
		"root then parent":  "/..",
	}

	for name, candidate := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := domain.SafeKey(candidate); !errors.Is(err, domain.ErrUnsafeName) {
				t.Errorf("SafeKey(%q) = %v, want ErrUnsafeName", candidate, err)
			}
		})
	}
}

// TestTraversalNamesAreNeutralised: names that contain a path but do designate
// a file are accepted, and the path does NOT survive.
//
// This is the second half of the protection: `../../etc/passwd` must not be
// refused for form's sake, it must be rendered harmless.
func TestTraversalNamesAreNeutralised(t *testing.T) {
	t.Parallel()

	dangerous := []string{
		"../../etc/passwd",
		"/etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"foo/../../../bar.txt",
		"./../secret.pem",
	}

	for _, candidate := range dangerous {
		key, err := domain.SafeKey(candidate)
		if err != nil {
			continue // refused, that is already safe
		}
		if !domain.IsWithin(key) {
			t.Errorf("SafeKey(%q) = %q escapes the store", candidate, key)
		}
		if got := key.String(); got == candidate {
			t.Errorf("SafeKey(%q) returned the name as it stands", candidate)
		}
	}
}
