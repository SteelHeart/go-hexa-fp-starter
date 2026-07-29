package generator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// validFeatureName constrains a business module's name.
//
// Strict `snake_case`, and that is a constraint of form, not of taste: the name
// becomes a DIRECTORY name, a key of `config/modules.yaml`, and — stripped of
// its underscores — a Go PACKAGE name. Accepting `My-Module` would produce a
// `MyModule` package, which `revive` refuses, inside a project whose barrier
// would fail on the very first command.
var validFeatureName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// FeaturePlan carries the verified values of a module creation.
//
// The exported fields are the ones the templates interpolate: `Module`, `Dir`,
// `Package`. The others are exported so tests can observe them without reaching
// inside the package.
type FeaturePlan struct {
	// Root is the root of the project where the module is created.
	Root string
	// Destination is the module's directory.
	Destination string
	// Module is the Go module path of the PROJECT, not of the business module.
	Module string
	// Dir is the directory name, in snake_case: `order_tracking`.
	Dir string
	// Package is the Go package name, without underscores: `ordertracking`.
	Package string
	// anchor carries the insertion point of arch-go's sealing rule.
	//
	// Located during PLANNING, written during execution: without that, an
	// unexpected arch-go.yml would fail the command after the module had already
	// been created, leaving an unguarded module on disk.
	anchor IsolationAnchor
}

// PlanFeature checks EVERYTHING before a single file is written.
func PlanFeature(name, root string) (FeaturePlan, error) {
	if !validFeatureName.MatchString(name) {
		return FeaturePlan{}, fmt.Errorf(
			"invalid module name %q — snake_case expected: `billing`, `order_tracking`", name)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FeaturePlan{}, fmt.Errorf("project path: %w", err)
	}
	module, err := ModulePathOf(absRoot)
	if err != nil {
		return FeaturePlan{}, err
	}

	destination := filepath.Join(absRoot, "internal", "modules", name)
	if occupied := EmptyDestination(destination); occupied != nil {
		return FeaturePlan{}, fmt.Errorf("the module already exists: %w", occupied)
	}

	anchor, err := FindIsolationAnchor(absRoot, name)
	if err != nil {
		return FeaturePlan{}, err
	}

	return FeaturePlan{
		Root:        absRoot,
		Destination: destination,
		Module:      module,
		Dir:         name,
		Package:     strings.ReplaceAll(name, "_", ""),
		anchor:      anchor,
	}, nil
}
