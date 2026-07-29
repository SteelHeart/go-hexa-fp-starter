package generator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ProjectReport reports on a successful generation.
//
// A type rather than a second return: `CreateProject` would then have three,
// which the `arch-go` rule refuses — and it is right, that is the lesson learned
// five times in this repository.
type ProjectReport struct {
	// Files is the number of files copied.
	Files int
	// GitInitialised says whether the repository and its hooks could be set up.
	GitInitialised bool
}

// CreateProject runs the generation, then VERIFIES it.
//
// Verification is part of the command, it is not left to the user: a generated
// project that does not build must say so itself, not wait for somebody else's
// first `go build`.
func CreateProject(ctx context.Context, p ProjectPlan) (ProjectReport, error) {
	files, err := TrackedFiles(ctx, p.Source)
	if err != nil {
		return ProjectReport{}, err
	}
	if copied := CopyProject(p, files); copied != nil {
		return ProjectReport{}, copied
	}

	remaining, err := VerifyNoTrace(p)
	if err != nil {
		return ProjectReport{}, err
	}
	if len(remaining) > 0 {
		return ProjectReport{}, fmt.Errorf(
			"the starter path remains in %d file(s) — the project would silently "+
				"depend on another repository:\n  %s\n\nAdd them to the rewrite, or "+
				"declare them in CitesSocleByHistory — never let them through",
			len(remaining), strings.Join(remaining, "\n  "))
	}

	if err := compile(ctx, p.Destination); err != nil {
		return ProjectReport{}, err
	}
	return ProjectReport{Files: len(files), GitInitialised: initGit(ctx, p.Destination) == nil}, nil
}

// compile exercises the generated project.
func compile(ctx context.Context, destination string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = destination
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("the generated project does not build — the generation is at "+
			"fault, not the project:\n%s", out)
	}
	return nil
}

// initGit sets up the repository AND the hooks path.
//
// The two go together: `git init` alone would leave the `commit-msg` hook inert,
// hence a brand-new project without its anti-accident net.
//
// Failure is not blocking — git may be absent — but it is RETURNED, not
// swallowed: the caller decides whether to say so, and `ProjectReport` carries
// it.
func initGit(ctx context.Context, destination string) error {
	// Two calls spelled out rather than a loop over variable arguments: `gosec`
	// refuses the latter, and it is right to — a command whose arguments come
	// from a variable is exactly the shape one does not want to skim over.
	steps := []*exec.Cmd{
		exec.CommandContext(ctx, "git", "init", "--quiet"),
		exec.CommandContext(ctx, "git", "config", "core.hooksPath", ".githooks"),
	}
	for _, cmd := range steps {
		cmd.Dir = destination
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git repository not initialised: %w\n%s", err, out)
		}
	}
	return nil
}
