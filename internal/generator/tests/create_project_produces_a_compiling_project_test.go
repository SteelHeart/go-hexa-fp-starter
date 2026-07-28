package tests

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestCreateProjectProducesACompilingProject exercises the WHOLE chain:
// tracked files, copy, rewrite, verification, compilation, git.
//
// # Why this test exists even though it shells out to `go build`
//
// Compiling the generated project is not a detail of `CreateProject`, it IS its
// contract: "a generated project that does not compile must say so itself,
// rather than wait for someone else's first build". Until now that contract was
// only exercised by `task ci:generateur`, which copies the REAL socle — several
// seconds, and unavailable to `go test`.
//
// The socle built here is deliberately tiny — a go.mod and one package — so the
// chain runs in well under a second while proving the same property.
func TestCreateProjectProducesACompilingProject(t *testing.T) {
	t.Parallel()

	socle := miniSocle(t)
	destination := filepath.Join(t.TempDir(), "generated")

	plan, err := generator.PlanProject(destination, targetModule, socle)
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	report, err := generator.CreateProject(t.Context(), plan)
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	if report.Files < 2 {
		t.Errorf("files copied = %d, the mini socle carries more than that", report.Files)
	}
	if !report.GitInitialised {
		t.Error("git was not initialised: the generated project has no commit-msg hook")
	}

	// The module path is rewritten…
	code := read(t, filepath.Join(destination, "internal", "greet", "greet.go"))
	if strings.Contains(code, socleModule) {
		t.Error("the socle path survives: the project would depend on another repository")
	}

	// …and the hooks path is actually configured, not merely `git init`-ed.
	// A repository without it leaves `commit-msg` inert — this repository has
	// already shipped two hooks that git ignored everywhere.
	hooks := gitConfig(t, destination, "core.hooksPath")
	if hooks != ".githooks" {
		t.Errorf("core.hooksPath = %q, want .githooks", hooks)
	}
}

// TestCreateProjectRefusesASocleThatIsNotARepository: `git ls-files` is the only
// definition of "what belongs to the socle" that never goes stale.
//
// The refusal must name the cause: a directory that merely LOOKS like the socle
// would otherwise produce an empty project, and the emptiness would be blamed on
// the generator.
func TestCreateProjectRefusesASocleThatIsNotARepository(t *testing.T) {
	t.Parallel()

	socle := t.TempDir()
	write(t, filepath.Join(socle, "go.mod"), "module "+socleModule+"\n\ngo 1.25.12\n", 0o600)

	plan, err := generator.PlanProject(filepath.Join(t.TempDir(), "out"), targetModule, socle)
	if err != nil {
		t.Fatalf("planning: %v", err)
	}

	if _, err := generator.CreateProject(t.Context(), plan); err == nil {
		t.Fatal("a socle without a git repository had to be refused")
	}
}

// miniSocle builds the smallest git repository that behaves like the socle:
// a module path to rewrite, one compiling package, and a tracked file.
func miniSocle(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module "+socleModule+"\n\ngo 1.25.12\n", 0o600)
	write(t, filepath.Join(root, "internal", "greet", "greet.go"),
		"// Package greet exists so that `go build ./...` has something to build.\n"+
			"package greet\n\n// Origin names the module this package came from.\n"+
			"const Origin = \""+socleModule+"\"\n", 0o600)

	for _, args := range [][]string{{"init", "--quiet"}, {"add", "-A"}} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

// gitConfig reads one configuration key of the generated repository.
func gitConfig(t *testing.T, root, key string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", "config", "--get", key)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// A missing key is not a test failure by itself: the assertion above
		// decides. Returning the empty string keeps the message readable.
		return ""
	}
	return strings.TrimSpace(string(out))
}
