package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestVerifyNoTraceKnowsHowToFail is the witness of the generation guard
// (ADR 013).
//
// Without it, "the rewrite covered everything" and "the check no longer looks at
// anything" are indistinguishable. And the failure mode is the worst possible:
// the project COMPILES, because Go resolves the import against the original
// socle while it is reachable. The symptom arrives weeks later, on another
// machine, as a package that cannot be found.
func TestVerifyNoTraceKnowsHowToFail(t *testing.T) {
	t.Parallel()

	destination := t.TempDir()
	plan := generator.ProjectPlan{
		Destination:  destination,
		SocleModule:  socleModule,
		TargetModule: targetModule,
	}

	// Success direction: nothing carries the socle path.
	write(t, filepath.Join(destination, "clean.go"), "package clean\n", 0o600)
	remaining, err := generator.VerifyNoTrace(plan)
	if err != nil {
		t.Fatalf("a clean project must pass: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("a clean project reports %v", remaining)
	}

	// Failure direction: an ordinary file keeps the socle path.
	write(t, filepath.Join(destination, "forgotten.go"),
		"package forgotten\n\nimport \""+socleModule+"/internal/pkg/fp\"\n", 0o600)

	remaining, err = generator.VerifyNoTrace(plan)
	if err != nil {
		t.Fatalf("VerifyNoTrace: %v", err)
	}
	if len(remaining) != 1 || remaining[0] != "forgotten.go" {
		t.Fatalf("a forgotten socle path must be reported, got %v", remaining)
	}

	// A file on the declared list does not count, even carrying the path.
	if removed := os.Remove(filepath.Join(destination, "forgotten.go")); removed != nil {
		t.Fatalf("cleanup: %v", removed)
	}
	write(t, filepath.Join(destination, "CLAUDE.md"), "voir https://"+socleModule+"/pull/1\n", 0o600)

	remaining, err = generator.VerifyNoTrace(plan)
	if err != nil {
		t.Fatalf("VerifyNoTrace: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("a file declared in CitesSocleByHistory must not be counted, got %v", remaining)
	}
}
