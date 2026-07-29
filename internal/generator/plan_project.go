package generator

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ProjectPlan carries the verified values of a project generation.
//
// A type rather than four parameters: the two-returns rule holds for inputs too,
// and four strings in a row get silently swapped.
type ProjectPlan struct {
	// Source is the root of the copied starter.
	Source string
	// Destination is the directory of the created project.
	Destination string
	// SocleModule is the starter's module path, the one being replaced.
	SocleModule string
	// TargetModule is the created project's module path.
	TargetModule string
}

// PlanProject checks EVERYTHING before a single file is written.
//
// Every step REFUSES rather than repairs: a generator silently patching up
// dubious input produces a project nobody knows the state of.
func PlanProject(destination, targetModule, source string) (ProjectPlan, error) {
	if targetModule == "" {
		return ProjectPlan{}, errors.New("--module is mandatory: a project without a module path does not build")
	}
	if !strings.Contains(targetModule, "/") {
		return ProjectPlan{}, fmt.Errorf(
			"--module=%q does not look like a module path (expected: host/organisation/name)", targetModule)
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		return ProjectPlan{}, fmt.Errorf("starter path: %w", err)
	}
	socleModule, err := ModulePathOf(absSource)
	if err != nil {
		return ProjectPlan{}, err
	}
	if socleModule == targetModule {
		return ProjectPlan{}, fmt.Errorf(
			"--module=%q is already the starter's module: nothing to rewrite", targetModule)
	}

	absDest, err := filepath.Abs(destination)
	if err != nil {
		return ProjectPlan{}, fmt.Errorf("destination path: %w", err)
	}
	if occupied := EmptyDestination(absDest); occupied != nil {
		return ProjectPlan{}, occupied
	}

	return ProjectPlan{
		Source:       absSource,
		Destination:  absDest,
		SocleModule:  socleModule,
		TargetModule: targetModule,
	}, nil
}
