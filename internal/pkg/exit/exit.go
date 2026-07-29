// Package exit holds a program's exit codes, following `sysexits.h`.
//
// # Why a convention rather than 0 and 1
//
// A command-line binary is called by other programs far more often than by
// humans: deployment scripts, `Makefile`s, scheduled jobs, CI jobs. Those
// callers do not read messages; they read the exit code, and they need it to
// decide whether to **retry**.
//
// With `1` for everything, a password that is too short and an unreachable
// database are indistinguishable: the script retries one — pointlessly,
// forever — or gives up on the other, when a second attempt would have done.
//
// # Why `sysexits.h` rather than a homegrown table
//
// Because it has existed since 1980, because it is the one used by `sendmail`,
// `git` and most Unix tooling, and because an operator seeing `78` already knows
// it means a configuration error. An invented table forces everyone to read our
// documentation in order to interpret our output.
//
// # This package has no dependencies
//
// Integer constants, nothing else. That is what lets it live in `internal/pkg/`
// and be used by any binary without dragging anything along.
package exit

// Exit codes, as defined by `sysexits.h` (BSD).
//
// Only the ones this starter knows how to produce are declared. Adding more
// "just in case" would suggest some path returns them while nothing emits them —
// the same mistake as declaring a driver that does not exist (ADR 014).
const (
	// OK: everything went fine.
	OK = 0

	// Usage: the command line is malformed — unknown option, missing argument.
	// The caller must fix its command, never retry.
	Usage = 64

	// DataErr: the supplied data is invalid. That is the user's fault, not the
	// service's: retrying identically will fail identically.
	DataErr = 65

	// Unavailable: a service we depend on is unreachable. This is the ONLY code
	// in this list that warrants a retry — and that is its whole point.
	Unavailable = 69

	// Software: an internal error. Retrying costs nothing but promises nothing;
	// what is needed is reading the logs.
	Software = 70

	// Config: the configuration is inconsistent or incomplete. Distinct from
	// `Usage`: the command was correct, the environment is not — so it is not up
	// to the caller to fix its command line.
	Config = 78

	// NoPerm: the operation is refused by a guard, and the refusal is
	// DELIBERATE. A `seed` in production returns it: neither a failure nor a
	// typo, an assumed "no".
	NoPerm = 77
)
