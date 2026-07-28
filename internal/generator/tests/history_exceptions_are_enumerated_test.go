package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestHistoryExceptionsAreEnumerated: the exception list is ENUMERATED, never
// guessed.
//
// These files carry the socle path in LINKS to its pull requests and issues, not
// in imports. Rewriting them would point the socle history at a repository that
// never carried it.
//
// The symmetric risk is worse: an exception that is too wide would leave a real
// import unrewritten, and the project would silently depend on another
// repository. Hence the negative cases below, which matter as much as the
// positive ones — in particular the two lookalikes, `documentation/adrien/` and
// a nested `documentation/adr/`.
func TestHistoryExceptionsAreEnumerated(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"CLAUDE.md":                             true,
		"documentation/process/REPRISE.md":      true,
		"documentation/adr/013-a-guard.md":      true,
		"documentation/adr/README.md":           true,
		"README.md":                             false,
		"documentation/process/NOMENCLATURE.md": false,
		"documentation/technique/pilotes.md":    false,
		"internal/config/config.go":             false,
		"cmd/server/main.go":                    false,
		"go.mod":                                false,
		"documentation/adrien/note.md":          false,
		"something/documentation/adr/fake.md":   false,
	}

	for path, want := range cases {
		if got := generator.CitesSocleByHistory(path); got != want {
			t.Errorf("CitesSocleByHistory(%q) = %v, want %v", path, got, want)
		}
	}
}
