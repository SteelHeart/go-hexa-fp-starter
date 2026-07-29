// Command hexa is the starter's command line tool.
//
// # A thin shell, and nothing else
//
// This file declares flags, routes to a subcommand, and prints. All the logic
// lives in `internal/generator`, for a reason as mechanical as it is
// architectural: Go forbids importing a `main` package, so code placed here
// can be tested neither as a black box, nor from `{package}/tests/` — the two
// locations `rules/tests.md` provides for (#96).
//
// # What `hexa` does today, and what it will do
//
// `hexa new` copies the starter and rewrites its module path. It is a
// TEMPLATE, not a library: the generated project OWNS all the code, and a fix
// to the starter does not propagate into it.
//
// That is a sequencing choice, not a target (ADR 015). No package of the
// starter is importable — everything lives under `internal/` — and nobody has
// ever built an application on it. Deciding now what becomes public would
// therefore be guesswork. `hexa new` is the only way to produce the
// application whose import list will MEASURE that boundary.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// version is injected at build time, as for the two other binaries.
var version = "dev"

const usage = `hexa — tool of the hexagonal starter

  hexa new <destination> --module <module/path>   creates a project
  hexa make:feature <module_name>                 creates a business module
  hexa version                                    prints the version

Examples:
  hexa new ./my-project --module github.com/impactone/facturation
  hexa make:feature order_tracking

Options of "new":
  --module   REQUIRED. The Go module path of the created project.
  --depuis   Root of the starter to copy (default: the current directory).
             It must be a git repository: it is the TRACKED files that define
             the starter, which rules out .git/, bin/ and .env.

Options of "make:feature":
  --dans     Root of the project in which to create the module (default: the
             current directory). The name is in snake_case: it becomes a
             directory, a config/modules.yaml key, and — without its
             underscores — a Go package.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		// To stderr, and without destroying the destination: a failure is
		// diagnosed by looking at what was written, not by guessing what is
		// missing.
		fmt.Fprintf(os.Stderr, "hexa: %v\n", err)
		os.Exit(1)
	}
}

// run routes to the subcommand.
func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "new":
		return commandNew(args[1:])
	case "make:feature":
		return commandFeature(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q — `hexa help` lists what exists", args[0])
	}
}

// commandNew creates a project from the starter.
func commandNew(args []string) error {
	set := flag.NewFlagSet("new", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	module := set.String("module", "", "Go module path of the created project (required)")
	from := set.String("depuis", ".", "root of the starter to copy")

	options, positional := generator.SplitArguments(args, generator.FlagsWithValue(set))
	if err := set.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positional) != 1 {
		return errors.New("usage: hexa new <destination> --module <module/path>")
	}

	plan, err := generator.PlanProject(positional[0], *module, *from)
	if err != nil {
		return fmt.Errorf("hexa new: %w", err)
	}
	report, err := generator.CreateProject(context.Background(), plan)
	if err != nil {
		return fmt.Errorf("hexa new: %w", err)
	}
	announceProject(plan, report)
	return nil
}

// commandFeature creates a business module in an existing project.
func commandFeature(args []string) error {
	set := flag.NewFlagSet("make:feature", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	in := set.String("dans", ".", "root of the project in which to create the module")

	options, positional := generator.SplitArguments(args, generator.FlagsWithValue(set))
	if err := set.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positional) != 1 {
		return errors.New("usage: hexa make:feature <module_name>")
	}

	plan, err := generator.PlanFeature(positional[0], *in)
	if err != nil {
		return fmt.Errorf("hexa make:feature: %w", err)
	}
	if created := generator.CreateFeature(context.Background(), plan); created != nil {
		return fmt.Errorf("hexa make:feature: %w", created)
	}
	announceFeature(plan)
	return nil
}

// announceProject prints what was created and what comes next.
func announceProject(plan generator.ProjectPlan, report generator.ProjectReport) {
	fmt.Printf("Project created in %s\n", plan.Destination)
	fmt.Printf("  module %s\n", plan.TargetModule)
	fmt.Printf("  files  %d\n", report.Files)
	if !report.GitInitialised {
		fmt.Fprint(os.Stderr, "hexa: git repository not initialised — run `git init` and "+
			"`git config core.hooksPath .githooks` by hand\n")
	}
	fmt.Print(`
Next:
  cd <destination>
  task init          # .env and tooling
  task check         # the quality gate, identical to the CI

The demonstration module ` + "`internal/modules/user_registration`" + ` is the
REFERENCE SLICE: its shape is the one to copy to write a business module.
Removing it requires removing its mounting from cmd/server.
`)
}

// announceFeature prints the lines left to write, and says why they are not
// added automatically.
func announceFeature(plan generator.FeaturePlan) {
	fmt.Printf("Module %s created in %s\n\n", plan.Dir, plan.Destination)
	fmt.Printf(`Three lines remain to be written, and they are NOT written for you:
mounting a module is a decision per binary (ADR 014), not a consequence of its
creation. A generator wiring everything on its own would produce mounted
modules nobody chose to mount.

1. internal/modules/catalog.go — make the module DECLARABLE

       %s.Catalog(),

2. config/modules.yaml — enable it

       %s:
         driver: memory

3. cmd/server/main.go — mount it

       module, err := %s.New(cfg.Modules.DriverOf(%s.Name), %s.Deps{
           GenerateID: ...,
           Now:        %s.SystemClock(),
       })

Then: task check
`, plan.Package, plan.Dir, plan.Package, plan.Package, plan.Package, plan.Package)
}
