package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// featureTemplates carries the anatomy of a business module.
//
// Embedded rather than read from disk: the `hexa` binary must work from
// anywhere, including installed outside the repository.
//
// The files carry the `.tmpl` suffix for a mechanical reason: named `.go`, they
// would be compiled along with the rest of the package, and a template does not
// compile — it holds `{{.Module}}`.
//
//go:embed all:templates/feature
var featureTemplates embed.FS

// featureTemplateRoot is the prefix to strip from the embedded paths.
const featureTemplateRoot = "templates/feature"

// CreateFeature renders the template, hardens the architecture, then EXERCISES.
func CreateFeature(ctx context.Context, p FeaturePlan) error {
	if err := RenderFeature(p); err != nil {
		return err
	}
	if err := DeclareIsolation(p.anchor, p.Dir); err != nil {
		return err
	}
	return proveFeature(ctx, p)
}

// RenderFeature writes the module's tree.
func RenderFeature(p FeaturePlan) error {
	walk := fs.WalkDir(featureTemplates, featureTemplateRoot,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			relative := strings.TrimPrefix(strings.TrimPrefix(path, featureTemplateRoot), "/")
			target := filepath.Join(p.Destination, strings.TrimSuffix(relative, ".tmpl"))
			return renderOne(path, target, p)
		})
	if walk != nil {
		return fmt.Errorf("rendering the module template: %w", walk)
	}
	return nil
}

// renderOne writes one file of the template, formatted.
func renderOne(path, target string, p FeaturePlan) error {
	rendered, err := render(path, p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, rendered, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	return nil
}

// render applies the template, then FORMATS the result if it is Go.
//
// # Why formatting lives here rather than in the templates
//
// A template cannot be `gofmt`-clean by construction: identifier widths change
// with the module name, so any hand-written alignment is wrong for every name
// but one. Measured — the first version produced three files that `go fmt`
// rewrote, and the `fmt` step of the generated project's barrier flagged them.
//
// `format.Source` removes the entire class: no future reshuffle of a template
// can produce badly formatted code any more. And a template turned
// syntactically wrong no longer writes itself silently — it refuses here, with
// its position, instead of failing later on a `go build` of the project.
func render(path string, p FeaturePlan) ([]byte, error) {
	raw, err := fs.ReadFile(featureTemplates, path)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", path, err)
	}
	// `missingkey=error`: a template referencing a missing key would write
	// `<no value>` into Go code. Better to refuse loudly.
	model, err := template.New(path).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("template %s is unreadable: %w", path, err)
	}

	var buffer bytes.Buffer
	if rendered := model.Execute(&buffer, p); rendered != nil {
		return nil, fmt.Errorf("rendering %s: %w", path, rendered)
	}
	if !strings.HasSuffix(path, ".go.tmpl") {
		return buffer.Bytes(), nil
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("template %s does not produce valid Go: %w", path, err)
	}
	return formatted, nil
}

// proveFeature builds the WHOLE project and runs ALL its tests.
//
// Verification is part of the command, it is not left to the user: a generated
// module that does not build must say so itself.
//
// The whole project rather than only the new module's packages, for two reasons.
// The first is substantive: a module that breaks the rest of the project is a
// faulty module, and only noticing it at the next `task check` would cast doubt
// on the wrong change. The second is mechanical — `gosec` rightly refuses a
// command one of whose arguments is a variable, and a motivated exemption would
// cost more than the wider check.
func proveFeature(ctx context.Context, p FeaturePlan) error {
	steps := []*exec.Cmd{
		exec.CommandContext(ctx, "go", "build", "./..."),
		exec.CommandContext(ctx, "go", "test", "./..."),
	}
	for _, cmd := range steps {
		cmd.Dir = p.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("the generated module does not pass `%s` — the generation "+
				"is at fault, not the module:\n%s", strings.Join(cmd.Args, " "), out)
		}
	}
	return nil
}
