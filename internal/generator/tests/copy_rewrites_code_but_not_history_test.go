package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

const (
	socleModule  = "github.com/example/socle"
	targetModule = "github.com/impactone/billing"
)

// TestCopyRewritesCodeButNotHistory covers the heart of the generator: what is
// copied, what is rewritten, what is not.
//
// Three properties, each of which has already cost this repository something:
//
//   - the module path is rewritten in code. A partial rewrite produces a project
//     that COMPILES — Go resolves the import against the original socle while it
//     is reachable — but silently depends on another repository;
//   - `CLAUDE.md` is NOT rewritten: its occurrences are links to historical pull
//     requests, which would point at a repository that never carried them;
//   - the executable bit survives. This repository committed both of its git
//     hooks as 100644: git ignored them everywhere, on every machine, and
//     nothing said so.
func TestCopyRewritesCodeButNotHistory(t *testing.T) {
	t.Parallel()

	socle := fakeSocle(t, socleModule)
	destination := filepath.Join(t.TempDir(), "generated")

	plan := generator.ProjectPlan{
		Source:       socle,
		Destination:  destination,
		SocleModule:  socleModule,
		TargetModule: targetModule,
	}

	files, err := generator.TrackedFiles(t.Context(), socle)
	if err != nil {
		t.Fatalf("TrackedFiles: %v", err)
	}
	if copied := generator.CopyProject(plan, files); copied != nil {
		t.Fatalf("CopyProject: %v", copied)
	}

	code := read(t, filepath.Join(destination, "internal", "code.go"))
	if strings.Contains(code, socleModule) {
		t.Error("the socle path survives in code: the project would depend on another repository")
	}
	if !strings.Contains(code, targetModule) {
		t.Error("the target path was not written")
	}

	if guard := read(t, filepath.Join(destination, "CLAUDE.md")); !strings.Contains(guard, socleModule) {
		t.Error("CLAUDE.md was rewritten: its links to the socle history are lost")
	}

	info, err := os.Stat(filepath.Join(destination, "tools", "guard.sh"))
	if err != nil {
		t.Fatalf("re-reading the script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("permissions = %v: a non-executable guard guards nothing", info.Mode().Perm())
	}

	remaining, err := generator.VerifyNoTrace(plan)
	if err != nil {
		t.Fatalf("VerifyNoTrace: %v", err)
	}
	if len(remaining) > 0 {
		t.Errorf("VerifyNoTrace rejects a correct project: %v", remaining)
	}
}

// fakeSocle builds a minimal git repository that looks like the socle.
func fakeSocle(t *testing.T, module string) string {
	t.Helper()

	root := t.TempDir()
	write(t, filepath.Join(root, "go.mod"), "module "+module+"\n\ngo 1.25.12\n", 0o600)
	write(t, filepath.Join(root, "internal", "code.go"),
		"package internal\n\nimport \""+module+"/internal/pkg/fp\"\n\nvar _ = fp.None[int]\n", 0o600)
	write(t, filepath.Join(root, "CLAUDE.md"),
		"Voir https://"+module+"/pull/42 pour l'historique.\n", 0o600)
	write(t, filepath.Join(root, "tools", "guard.sh"), "#!/usr/bin/env sh\nexit 0\n", 0o750)

	for _, args := range [][]string{{"init", "--quiet"}, {"add", "-A"}} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}
