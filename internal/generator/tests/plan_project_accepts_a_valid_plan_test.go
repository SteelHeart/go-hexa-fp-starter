package tests

import (
	"path/filepath"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// TestPlanProjectAcceptsAValidPlan: the positive direction, without which the
// refusals prove nothing.
//
// A PlanProject that refused EVERYTHING would pass every refusal test. This is
// the same asymmetry ADR 013 names for guards: checking that something fails
// says nothing until you have checked that it succeeds.
func TestPlanProjectAcceptsAValidPlan(t *testing.T) {
	t.Parallel()

	socle := t.TempDir()
	write(t, filepath.Join(socle, "go.mod"), "module github.com/example/socle\n\ngo 1.25.12\n", 0o600)
	destination := filepath.Join(t.TempDir(), "new-project")

	plan, err := generator.PlanProject(destination, "github.com/impactone/billing", socle)
	if err != nil {
		t.Fatalf("a valid plan must be accepted: %v", err)
	}

	if plan.SocleModule != "github.com/example/socle" {
		t.Errorf("socle module read = %q", plan.SocleModule)
	}
	if plan.TargetModule != "github.com/impactone/billing" {
		t.Errorf("target module = %q", plan.TargetModule)
	}
	if !filepath.IsAbs(plan.Source) || !filepath.IsAbs(plan.Destination) {
		t.Error("paths must be absolute: a relative path would depend on the working " +
			"directory at copy time, not at call time")
	}
}
