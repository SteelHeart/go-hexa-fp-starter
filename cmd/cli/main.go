// Command cli exposes the starter on the COMMAND LINE surface.
//
// # Why this binary exists
//
// To demonstrate the starter's property no. 2 — *the number of frontends is a
// non-issue* — rather than to state it. With the HTTP surface alone it was
// written; with this one it is **measured**: the impact map of adding it
// touches nothing but `adapters/primary/`, and not one line of `domain/`,
// `ports/` or `application/`.
//
// The `register` command calls EXACTLY the same port as `POST /v1/users`. The
// module does not know which of the two surfaces is calling it.
//
// # It is a composition root, like `cmd/server` and `cmd/worker`
//
// It is allowed to know everything (ADR 004), and it reads the SAME
// configuration: a module enabled here is enabled over there, with the same
// driver. An administration binary with a configuration of its own would end
// up acting on a store other than the one it claims to administer.
//
// # Exit codes
//
// `sysexits.h`, because a command line binary is called by scripts more often
// than by humans, and because a script reads the exit code to decide whether
// to RETRY. See internal/pkg/exit.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

func main() { os.Exit(start()) }

// start installs the signal handling and returns the exit code.
//
// Separated from `main` because `os.Exit` TRIGGERS NO `defer`: placing them in
// `main` around an `os.Exit` amounts to not placing them at all. Here the
// function returns an integer, all its `defer`s run, and `main` does nothing
// more than exit.
func start() int {
	// NotifyContext from the outset: a Ctrl+C during a `seed` must interrupt
	// cleanly, not leave half the accounts created without anyone knowing
	// which ones.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, os.Args[1:])
}

// run dispatches the subcommand and returns its exit code.
//
// It returns an INTEGER rather than calling `os.Exit` itself: that is what
// makes the dispatch verifiable by a test, and what guarantees that the
// `defer`s placed upstream run — `os.Exit` skips them all, including the
// closing of a pool.
func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		usage()
		return exit.Usage
	}

	switch args[0] {
	case "register":
		return commandRegister(ctx, args[1:])
	case "seed":
		return commandSeed(ctx, args[1:])
	case "health":
		return commandHealth(ctx)
	case "-h", "--help", "help":
		usage()
		return exit.OK
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", args[0])
		usage()
		return exit.Usage
	}
}

// usage describes what this binary can do.
//
// `migrate:status` is NOT listed, and that is a decision: the migration status
// is already returned by `goose`, which the `Taskfile` and the CI call.
// Reimplementing it here would create a second source of truth on the state of
// the schema — and the day the two diverged, it is the lying one that would be
// believed.
func usage() {
	fmt.Fprint(os.Stderr, `usage: hexa-cli <command> [options]

  register --email <address>   register an account; the password is read from stdin
  seed --profile <dev|demo>    seed the starter THROUGH THE USE CASES — never in production
  health                       check that this binary could start

The migration status is returned by `+"`task migrate:status`"+`, not by this command:
a second source of truth on the schema is one source of truth too many.
`)
}
