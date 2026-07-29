package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// TestTheCLICallsTheSamePortAsHTTP is the WITNESS of issue #8.
//
// # What it demonstrates
//
// That the surface is a detail. This command calls `mod.Register` — the same
// port as `POST /v1/users` — and obtains the same outcome, produced by the same
// domain code. The module does not know which of the two surfaces is calling it,
// and nothing had to be added to it for it to serve both.
//
// This is the starter's property no. 2, *the number of frontends is a
// non-issue*. With a single surface it was stated; with two, it is measured.
//
// # What the test verifies, precisely
//
// The identifier, the NORMALISED address and the status come out on STANDARD
// output, separated by tabs — a line a script can cut up. And the exit code is
// zero.
func TestTheCLICallsTheSamePortAsHTTP(t *testing.T) {
	t.Parallel()

	cmd, captured := newCommand(t, secret+"\n")
	code := cmd.Register(context.Background(), []string{"--email", "  Alice@Example.COM "})

	if code != exit.OK {
		t.Fatalf("want %d, got %d — err: %s", exit.OK, code, captured.err.String())
	}

	line := strings.TrimSpace(captured.out.String())
	fields := strings.Split(line, "\t")
	if len(fields) != 3 {
		t.Fatalf("the output must carry 3 tab-separated fields, got %q", line)
	}
	if fields[1] != email {
		t.Fatalf("the address must be NORMALISED by the domain: %q instead of %q", fields[1], email)
	}
	if fields[2] != "pending" {
		t.Fatalf("the account is born `pending`, got %q", fields[2])
	}
}

// TestTheSecretNeverLeaksIntoTheOutput guards both streams.
//
// A secret copied into the output would end up in the log of every script that
// captures the command — and an administration script is always captured. The
// digest must not come out either: it is broken offline.
func TestTheSecretNeverLeaksIntoTheOutput(t *testing.T) {
	t.Parallel()

	cmd, captured := newCommand(t, secret+"\n")
	if code := cmd.Register(context.Background(), []string{"--email", email}); code != exit.OK {
		t.Fatalf("registration: code %d — %s", code, captured.err.String())
	}

	for name, content := range map[string]string{
		"stdout": captured.out.String(),
		"stderr": captured.err.String(),
	} {
		if strings.Contains(content, secret) {
			t.Errorf("the CLEAR-TEXT secret leaks on %s: %q", name, content)
		}
		if strings.Contains(content, "digest:") {
			t.Errorf("the digest leaks on %s: %q", name, content)
		}
	}
}

// TestErrorsGoToStderrNeverToStdout: the two streams do not get mixed up.
//
// # Why this is a property and not a preference
//
// The standard output of a CLI is DATA: a downstream script cuts it up. Writing
// an error message there breaks that cutting up — and breaks it on the day of
// the incident, when an account is refused, that is to say at the worst possible
// moment.
//
// The defect is invisible as long as one looks at a terminal, where the two
// streams are displayed in the same place. It only appears when redirecting one
// without the other.
func TestErrorsGoToStderrNeverToStdout(t *testing.T) {
	t.Parallel()

	cmd, captured := newCommand(t, "too-short\n")
	code := cmd.Register(context.Background(), []string{"--email", email})

	if code != exit.DataErr {
		t.Fatalf("a secret that is too short is invalid data: want %d, got %d", exit.DataErr, code)
	}
	if captured.out.Len() != 0 {
		t.Fatalf("nothing must come out on stdout on failure, got %q", captured.out.String())
	}
	if captured.err.Len() == 0 {
		t.Fatal("the error must be written on stderr")
	}
	if !strings.Contains(captured.err.String(), "field:") {
		t.Fatalf("the faulty field must be named: %q", captured.err.String())
	}
}
