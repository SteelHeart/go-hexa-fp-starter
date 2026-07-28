package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestPlanProjectRefusesBeforeWriting: every check happens BEFORE the first
// write.
//
// A generator that discovers a problem halfway through the copy leaves a
// half-written destination that nobody knows whether to use. The refusal must
// therefore be complete and early.
//
// Each case is a refusal, and each refusal must NAME what is wrong: a message
// that only says "error" sends people searching at random.
func TestPlanProjectRefusesBeforeWriting(t *testing.T) {
	t.Parallel()

	socle := t.TempDir()
	write(t, filepath.Join(socle, "go.mod"), "module github.com/example/socle\n\ngo 1.25.12\n", 0o600)

	busy := t.TempDir()
	write(t, filepath.Join(busy, "already-there.txt"), "x", 0o600)

	cases := map[string]struct {
		destination string
		module      string
		source      string
		reason      string
	}{
		"missing module": {
			destination: filepath.Join(t.TempDir(), "new"),
			module:      "",
			source:      socle,
			reason:      "obligatoire",
		},
		"module without a slash": {
			destination: filepath.Join(t.TempDir(), "new"),
			module:      "billing",
			source:      socle,
			reason:      "chemin de module",
		},
		"module identical to the socle": {
			destination: filepath.Join(t.TempDir(), "new"),
			module:      "github.com/example/socle",
			source:      socle,
			reason:      "déjà le module du socle",
		},
		"socle without go.mod": {
			destination: filepath.Join(t.TempDir(), "new"),
			module:      "github.com/example/target",
			source:      t.TempDir(),
			reason:      "go.mod",
		},
		"destination not empty": {
			destination: busy,
			module:      "github.com/example/target",
			source:      socle,
			reason:      "n'est pas vide",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := generator.PlanProject(tc.destination, tc.module, tc.source)
			if err == nil {
				t.Fatal("an invalid plan must be refused BEFORE any write")
			}
			if !strings.Contains(err.Error(), tc.reason) {
				t.Errorf("message = %q, it must contain %q to be actionable", err, tc.reason)
			}
		})
	}
}
