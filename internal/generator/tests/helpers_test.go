// Package tests holds the BLACK BOX tests of the generator: they only use its
// public API, exactly like `cmd/hexa` does.
//
// # Why these tests used to live elsewhere, and badly
//
// They lived at the root of `cmd/hexa`, in `package main`. That was the only
// possible location — Go forbids importing a `main` package — but it was neither
// the black box nor `internal_test.go`, the only two `rules/tests.md` provides
// for. Moving the logic into `internal/generator` made this location possible
// (#96).
//
// One file per test, named after it, shared helpers here and nowhere else.
package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// archGoWithOneRule is the minimal content `PlanFeature` expects.
//
// It carries comments ON BOTH SIDES of the insertion point: that is what allows
// checking the insertion does not wipe them.
const archGoWithOneRule = `dependenciesRules:
  # A comment that must SURVIVE the insertion.
  - package: "**.internal.modules.user_registration.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.!(user_registration).**"

  # A second comment, after the anchor point.
  - package: "**.internal.core.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.**"
`

// fakeProject builds the minimal structure a feature plan requires.
//
// Building it here rather than generating a real project keeps these tests in
// milliseconds: a real generation runs `go build` and `go test`.
func fakeProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write(t, filepath.Join(root, "go.mod"), "module github.com/example/project\n\ngo 1.25.12\n", 0o600)
	write(t, filepath.Join(root, "arch-go.yml"), archGoWithOneRule, 0o600)
	return root
}

func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(content)
}
