package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestPlanFeatureRefusesBeforeWriting: every refusal happens during PLANNING,
// therefore before a single file is written.
//
// This is what separates a generator from a script: a command that fails halfway
// leaves a half-created module, and its author has to guess what to clean up.
// Here, either everything is written or nothing is.
func TestPlanFeatureRefusesBeforeWriting(t *testing.T) {
	t.Parallel()

	root := fakeProject(t)

	cases := map[string]string{
		"upper case":          "OrderTracking",
		"dash":                "order-tracking",
		"leading digit":       "1billing",
		"doubled underscore":  "order__tracking",
		"trailing underscore": "order_",
		"empty":               "",
		"disguised path":      "../escape",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := generator.PlanFeature(input, root); err == nil {
				t.Errorf("PlanFeature(%q) had to refuse", input)
			}
		})
	}
}

// TestPlanFeatureDerivesTheGoPackage: a valid name yields the expected directory
// and package name.
//
// The derivation is not cosmetic: `order_tracking` is correct as a DIRECTORY
// name and as a configuration key, but `revive` refuses a Go package carrying an
// underscore. Both forms therefore coexist, and the generator keeps them
// consistent.
func TestPlanFeatureDerivesTheGoPackage(t *testing.T) {
	t.Parallel()

	plan, err := generator.PlanFeature("order_tracking", fakeProject(t))
	if err != nil {
		t.Fatalf("a valid name had to pass: %v", err)
	}

	if plan.Dir != "order_tracking" {
		t.Errorf("Dir = %q, want the original snake_case", plan.Dir)
	}
	if plan.Package != "ordertracking" {
		t.Errorf("Package = %q, want it without underscores", plan.Package)
	}
	if plan.Module != "github.com/example/project" {
		t.Errorf("Module = %q, want the one from go.mod", plan.Module)
	}
	if !strings.HasSuffix(plan.Destination, filepath.Join("internal", "modules", "order_tracking")) {
		t.Errorf("Destination = %q, want it under internal/modules", plan.Destination)
	}
}

// TestPlanFeatureRefusesAnExistingModule: an existing module is never
// overwritten.
//
// Deny by default: overwriting is irreversible, and the generator has no way to
// know what the previous module contained.
func TestPlanFeatureRefusesAnExistingModule(t *testing.T) {
	t.Parallel()

	root := fakeProject(t)
	write(t, filepath.Join(root, "internal", "modules", "billing", "module.go"), "package billing\n", 0o600)

	if _, err := generator.PlanFeature("billing", root); err == nil {
		t.Error("an existing module had to make the command refuse")
	}
}
