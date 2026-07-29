package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestDeclareIsolationKeepsTheRestByteForByte: the isolation rule is inserted,
// and the rest of the file survives untouched.
//
// `arch-go.yml` is almost entirely made of comments explaining WHY each rule
// exists. A YAML round trip would erase every one of them — which is why the
// insertion is textual and not structural.
func TestDeclareIsolationKeepsTheRestByteForByte(t *testing.T) {
	t.Parallel()

	root := fakeProject(t)
	path := filepath.Join(root, "arch-go.yml")
	before := read(t, path)

	anchor, err := generator.FindIsolationAnchor(root, "billing")
	if err != nil {
		t.Fatalf("anchor not found: %v", err)
	}
	if err := generator.DeclareIsolation(anchor, "billing"); err != nil {
		t.Fatalf("insertion: %v", err)
	}

	after := read(t, path)

	if !strings.Contains(after, `- package: "**.internal.modules.billing.**"`) {
		t.Error("the new module's rule is missing")
	}
	if !strings.Contains(after, `- "**.internal.modules.!(billing).**"`) {
		t.Error("the new module's exclusion is missing")
	}
	for _, kept := range []string{
		"# A comment that must SURVIVE the insertion.",
		"# A second comment, after the anchor point.",
		`- package: "**.internal.core.**"`,
	} {
		if !strings.Contains(after, kept) {
			t.Errorf("the insertion lost %q", kept)
		}
	}
	if len(after) <= len(before) {
		t.Error("the file had to grow")
	}
}

// TestAProjectWithoutIsolationRuleIsRefused: without an anchor, the command
// REFUSES instead of creating a module outside every guard.
//
// # Why this refusal is the heart of the matter
//
// The isolation rule NAMES each module explicitly. A module created without it
// is covered by NO rule: it could import any other module, and `arch-go` would
// report "100% compliance" — because it has nothing to say about a module nobody
// told it about.
//
// This is the eleventh time this repository meets that shape: a silent guard is
// indistinguishable from a satisfied one. Hence the refusal, rather than a
// warning nobody would read.
func TestAProjectWithoutIsolationRuleIsRefused(t *testing.T) {
	t.Parallel()

	root := fakeProject(t)
	write(t, filepath.Join(root, "arch-go.yml"), "dependenciesRules: []\n", 0o600)

	_, err := generator.FindIsolationAnchor(root, "billing")
	if err == nil {
		t.Fatal("an arch-go.yml without an isolation rule had to make the command refuse")
	}
	// Refusing without saying what to do turns a guard into a dead end.
	if !strings.Contains(err.Error(), `- package: "**.internal.modules.billing.**"`) {
		t.Errorf("the refusal must dictate the rule to write, got:\n%v", err)
	}
}

// TestAnAlreadyDeclaredModuleIsRefused: a module is never declared twice.
//
// Without this refusal, running the command again would stack identical rules,
// and `arch-go` would end up reading a file nobody understands.
func TestAnAlreadyDeclaredModuleIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := generator.FindIsolationAnchor(fakeProject(t), "user_registration"); err == nil {
		t.Error("an already declared module had to make the command refuse")
	}
}

// TestAProjectWithoutArchGoIsRefused: a missing file is an error, not a
// permission.
func TestAProjectWithoutArchGoIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := generator.FindIsolationAnchor(t.TempDir(), "billing"); err == nil {
		t.Error("a project without arch-go.yml had to make the command refuse")
	}
}
