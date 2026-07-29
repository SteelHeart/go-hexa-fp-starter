package tests

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestRenderFeatureWritesTheWholeAnatomy: the created module carries EVERY
// directory of the anatomy, and every Go file is formatted.
//
// # Why completeness is checked, not just compilation
//
// `documentation/AMORCAGE.md` puts it in one sentence that applies here: *any missing directory
// would be reproduced as "not necessary"*. A template that forgot `drivers/`
// would produce modules without an interchangeable driver, and nobody would
// notice until the day one had to be swapped.
//
// Formatting is checked for a measured reason: a template cannot be gofmt-clean
// by construction, since identifier widths depend on the module name. The first
// version produced three files that `go fmt` rewrote — a red step in the
// generated project.
func TestRenderFeatureWritesTheWholeAnatomy(t *testing.T) {
	t.Parallel()

	plan, err := generator.PlanFeature("order_tracking", fakeProject(t))
	if err != nil {
		t.Fatalf("planning: %v", err)
	}
	if err := generator.RenderFeature(plan); err != nil {
		t.Fatalf("rendering: %v", err)
	}

	expected := []string{
		"module.go",
		"catalog.go",
		filepath.Join("domain", "errors.go"),
		filepath.Join("domain", "reference.go"),
		filepath.Join("domain", "record.go"),
		filepath.Join("ports", "ports.go"),
		filepath.Join("application", "create_record.go"),
		filepath.Join("drivers", "memory", "memory.go"),
	}
	for _, relative := range expected {
		if _, err := os.Stat(filepath.Join(plan.Destination, relative)); err != nil {
			t.Errorf("the anatomy is incomplete, %s is missing: %v", relative, err)
		}
	}

	for _, dir := range []string{"domain/tests", "application/tests", "tests"} {
		entries, err := os.ReadDir(filepath.Join(plan.Destination, filepath.FromSlash(dir)))
		if err != nil || len(entries) == 0 {
			t.Errorf("%s must hold black-box tests (%v)", dir, err)
		}
	}

	checkFormatting(t, plan.Destination)
}

// TestRenderFeatureRewritesTheModulePath: no file keeps the socle's module path.
//
// This is the defect that would make the module unusable elsewhere, and it is
// silent: the project would compile, pulling code from the socle repository
// rather than its own.
func TestRenderFeatureRewritesTheModulePath(t *testing.T) {
	t.Parallel()

	plan, err := generator.PlanFeature("billing", fakeProject(t))
	if err != nil {
		t.Fatalf("planning: %v", err)
	}
	if err := generator.RenderFeature(plan); err != nil {
		t.Fatalf("rendering: %v", err)
	}

	module := read(t, filepath.Join(plan.Destination, "module.go"))

	if !strings.Contains(module, "github.com/example/project/internal/modules/billing/domain") {
		t.Error("imports must carry the PROJECT's module path")
	}
	if strings.Contains(module, "go-hexa-fp-starter") {
		t.Error("the socle's module path survived the rendering")
	}
	if !strings.Contains(module, "package billing") {
		t.Error("the package must carry the name derived from the module")
	}
	if strings.Contains(module, "<no value>") {
		t.Error("a template key was left unresolved")
	}
}

// checkFormatting fails if any generated Go file is not already gofmt-clean.
func checkFormatting(t *testing.T, root string) {
	t.Helper()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		raw := read(t, path)
		want, err := format.Source([]byte(raw))
		if err != nil {
			t.Errorf("%s is not valid Go: %v", path, err)
			return nil
		}
		if string(want) != raw {
			t.Errorf("%s is not formatted — `go fmt` would rewrite it", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking: %v", err)
	}
}
