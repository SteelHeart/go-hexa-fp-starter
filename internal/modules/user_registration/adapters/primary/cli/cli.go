// Package cli exposes the registration module on the COMMAND LINE surface.
//
// # This package is the demonstration of the starter's property no. 2
//
// *The number of frontends is a non-issue.* It is not demonstrated by writing
// it down: it is demonstrated by adding a surface and observing that not a line
// of `domain/`, `ports/` or `application/` moves. That is the case here — this
// package calls **the same port** as the HTTP surface, `mod.Register`, and the
// module does not know which of the two is calling it.
//
// With a single surface, the property was stated. With two, it is measured: the
// impact map of this PR only touches `adapters/primary/`.
//
// # A surface is a TRANSLATOR, never a place for business
//
// This package does three things: read arguments, call the use case, translate
// the outcome into output and an EXIT CODE. It validates nothing itself — the
// domain already does, and duplicating the validation guarantees that one day
// the two will diverge, to the detriment of the one that speaks to the user.
//
// # Map of the files
//
//	cli.go        the command, its streams, and the translation of errors
//	register.go   `register` — register an account
package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Command carries what the surface needs in order to answer.
//
// # Why a structure rather than parameters
//
// `Run(ctx, mod, args, out, err)` makes five arguments, and the architecture
// rule allows four. It is right here for a reason specific to the subject: `out`
// and `err` have the same type, so swapping them compiles — and a binary that
// writes its errors on standard output silently breaks every script that
// redirects one without the other.
//
// # The streams are INJECTED, never `os.Stdout`
//
// That is what makes this surface testable without launching a process: a test
// gives it two buffers and reads what comes out. A package that wrote directly
// to `os.Stdout` would only be verifiable by capturing the process descriptors —
// therefore not in parallel, and not in a readable way.
type Command struct {
	Module userregistration.Module
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
}

// readSecret reads the password from the provided input.
//
// # Why standard input and not a flag
//
// A `--password` shows up in `ps`, in the shell history, and in command audit
// logs — three places nobody thinks to purge. It is the most commonplace mistake
// of an administration tool, and it costs nothing to avoid.
//
// # NON-GUARANTEE: terminal echo is NOT turned off
//
// In interactive use, the password is displayed while being typed. Turning it
// off requires manipulating the terminal (`golang.org/x/term`), therefore
// telling a terminal apart from a pipe — and a binary that behaves differently
// depending on whether it is piped or not is a binary whose scripts break at the
// first change of environment.
//
// The intended use is `printf '%s' "$SECRET" | hexa-cli register --email …`,
// where the question does not arise. The weakness is written down rather than
// left unsaid.
func (c Command) readSecret() (string, error) {
	if c.In == nil {
		return "", errNoInput
	}

	// Byte by byte reading, WITHOUT a buffer, and that is a decision.
	//
	// `bufio.Reader` reads AHEAD: it fills its buffer well beyond the newline,
	// and everything it has absorbed is lost for the next reader. A secret
	// reader must consume its line and nothing else — otherwise it swallows
	// what follows on the input, and the caller only discovers it by observing
	// that their data has vanished.
	//
	// Found by a test that called `Register` twice: the second call read an
	// empty input although it did contain a second secret. A password is short:
	// the cost of byte by byte reading is nil.
	var secret strings.Builder
	oneByte := make([]byte, 1)
	for {
		n, err := c.In.Read(oneByte)
		if n > 0 {
			if oneByte[0] == '\n' {
				break
			}
			secret.WriteByte(oneByte[0])
		}
		if errors.Is(err, io.EOF) {
			// An input without a trailing newline is LEGITIMATE: `printf`
			// without `\n` produces one, and it is the safest form for a
			// secret.
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading the secret: %w", err)
		}
	}

	line := strings.TrimRight(secret.String(), "\r")
	if line == "" {
		return "", errEmptySecret
	}
	return line, nil
}

// Errors of the surface itself, distinct from those of the domain.
var (
	errNoInput     = errors.New("no standard input: the secret is read from stdin")
	errEmptySecret = errors.New("empty secret: nothing was read from standard input")
)

// codeFor translates a domain error into a `sysexits` exit code.
//
// # The `switch` is EXHAUSTIVE, and the linter guards it
//
// Adding an error code to the domain forces a decision, HERE, about what an
// automated caller sees of it. Without that constraint, a fresh code would
// silently translate into "internal error" — and a script would stop retrying a
// transient outage, or retry an invalid input forever.
//
// It is the same requirement as `statusFor` on the HTTP side, for the same
// reason: the distinction serves to decide about a RETRY, and it cannot be
// guessed from a message.
//
// The `default` stays despite exhaustiveness: it covers the zero value and any
// string fabricated by a conversion. Deny by default — the unknown is a
// breakdown, never a success.
func codeFor(err domain.Error) int {
	switch err.Code {
	case domain.CodeInvalidEmail, domain.CodeWeakPassword:
		// Invalid data: retrying identically will fail the same way.
		return exit.DataErr

	case domain.CodeEmailAlreadyExists:
		// The STATE of the server opposes it, not the command. `DataErr` and
		// not `Unavailable`: retrying will change nothing as long as the
		// account exists.
		return exit.DataErr

	case domain.CodeUnavailable:
		// The ONLY code that authorises a retry, and the whole usefulness of
		// the table.
		return exit.Unavailable

	case domain.CodeInternal:
		return exit.Software

	default:
		return exit.Software
	}
}
