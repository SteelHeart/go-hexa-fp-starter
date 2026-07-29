// Package generator holds everything the `hexa` command does, minus the
// dispatching.
//
// # Why this logic does not live in `cmd/hexa`
//
// It did, and that was a measurable mistake: `cmd/hexa` is a `package main`, and
// **Go forbids importing a `main` package**. Its tests could therefore be
// neither black box nor inside `{package}/tests/` — the two requirements of
// `rules/tests.md`. Ten test files had piled up at the root of the package,
// outside the two locations the rule provides for (#96).
//
// The move fixes three things at once:
//
//   - tests become possible **through the public API**, inside `tests/`;
//   - `covergate` finally COUNTS them, `cmd/` being outside the unit scope;
//   - `cmd/hexa/main.go` becomes again what a composition root must be — a thin
//     shell declaring flags and dispatching (ADR 004).
//
// # This package knows NOTHING about the application
//
// It manipulates files, not modules. It imports neither `internal/core`, nor
// `internal/modules`, nor `internal/infrastructure` — enforced by `arch-go`.
// That is what will make it extractable the day the starter becomes an
// importable library (ADR 015).
//
// # File map
//
//	arguments.go        argument sorting, before `flag`
//	plan_project.go     PlanProject — checks EVERYTHING before writing
//	create_project.go   CreateProject — copies, verifies, builds, inits git
//	copy_project.go     TrackedFiles, CopyProject
//	verify_project.go   VerifyNoTrace — no leftover of the starter's path
//	plan_feature.go     PlanFeature
//	create_feature.go   CreateFeature, RenderFeature
//	isolation.go        the `arch-go` sealing rule of the created module
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EmptyDestination refuses to write into a non-empty directory.
//
// Deny by default: overwriting existing files is irreversible, and a generator
// has no way of knowing what mattered to the user.
func EmptyDestination(path string) error {
	entries, err := os.ReadDir(path)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("reading the destination: %w", err)
	case len(entries) > 0:
		return fmt.Errorf("%s is not empty — pick a fresh directory", path)
	}
	return nil
}

// ModulePathOf reads the module path declared by a Go project.
func ModulePathOf(root string) (string, error) {
	// Root supplied by the caller, and that is intended: it names THEIR project.
	//nolint:gosec // root named by the caller, fixed file name
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("%s has no go.mod — is this really the project root? %w", root, err)
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "module "); found {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no `module` directive in %s/go.mod", root)
}
