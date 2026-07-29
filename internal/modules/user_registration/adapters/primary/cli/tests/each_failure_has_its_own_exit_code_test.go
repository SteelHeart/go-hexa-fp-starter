package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// TestEachFailureHasItsOwnExitCode: the exit code serves to decide about a
// RETRY.
//
// # Why this is not cosmetic
//
// A command line binary is called by scripts far more often than by humans.
// Those callers do not read the messages; they read the exit code.
//
// With `1` for everything, a password that is too short and an unreachable
// database are indistinguishable: the script retries one — pointlessly, for
// ever — or gives up on the other, when a second attempt would have been enough.
//
// The table is the one from `sysexits.h`, dating back to 1980, so that an
// operator who sees `78` already knows it is about configuration without reading
// our documentation.
func TestEachFailureHasItsOwnExitCode(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args  []string
		input string
		code  int
	}{
		"invalid address":  {[]string{"--email", "not-an-address"}, secret + "\n", exit.DataErr},
		"secret too short": {[]string{"--email", email}, "short\n", exit.DataErr},
		"missing address":  {[]string{}, secret + "\n", exit.Usage},
		"unknown flag":     {[]string{"--e-mail", email}, secret + "\n", exit.Usage},
		"empty secret":     {[]string{"--email", email}, "\n", exit.Usage},
		"empty input":      {[]string{"--email", email}, "", exit.Usage},
	}

	for name, tc := range cases {
		cmd, captured := newCommand(t, tc.input)
		code := cmd.Register(context.Background(), tc.args)
		if code != tc.code {
			t.Errorf("%s: want %d, got %d — %s", name, tc.code, code, captured.err.String())
		}
	}
}

// TestATakenAddressIsNotRetryable tells the state of the server apart from a
// breakdown.
//
// An already taken address returns `DataErr` and not `Unavailable`: the request
// is well formed, but retrying will change nothing as long as the account
// exists. It is exactly the distinction the HTTP surface makes between 409 and
// 503, expressed here through an exit code.
func TestATakenAddressIsNotRetryable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd, captured := newCommand(t, secret+"\n"+secret+"\n")

	if code := cmd.Register(ctx, []string{"--email", email}); code != exit.OK {
		t.Fatalf("first registration: code %d — %s", code, captured.err.String())
	}
	if code := cmd.Register(ctx, []string{"--email", email}); code != exit.DataErr {
		t.Fatalf("address already taken: want %d, got %d", exit.DataErr, code)
	}
	if strings.Contains(captured.err.String(), "digest:") {
		t.Fatalf("the digest leaks into the conflict message: %q", captured.err.String())
	}
}

// TestSeedRefusesAnUnknownProfile: deny by default, right down to a flag.
//
// Falling back on the smallest profile would create three accounts where
// twenty-five were expected — and the defect would only be seen the moment a
// paginated list stopped paginating, that is to say during a demonstration.
func TestSeedRefusesAnUnknownProfile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for _, profile := range []string{"perf", "prod", "DEV", ""} {
		cmd, captured := newCommand(t, "")
		code := cmd.Seed(ctx, []string{"--profile", profile})
		if code != exit.Usage {
			t.Errorf("profile %q: want %d, got %d", profile, exit.Usage, code)
		}
		if captured.out.Len() != 0 {
			t.Errorf("profile %q: no account must be created, got %q", profile, captured.out.String())
		}
	}
}

// TestSeedGoesThroughTheUseCaseNotThroughSQL guards the seeding path.
//
// # The property that matters
//
// The generated accounts are in every respect the ones the HTTP surface would
// have created: address normalised by the domain, `pending` status, hashed
// password. A data set fabricated in SQL would produce accounts the code would
// never have let come into existence — and a test relying on them would validate
// a state production cannot reach.
//
// The test establishes it through the only path observable from the outside: the
// output of the seeding carries the same three fields as `register`, and the
// second pass returns a CONFLICT — so the accounts really do exist in the store,
// they were not invented.
func TestSeedGoesThroughTheUseCaseNotThroughSQL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cmd, captured := newCommand(t, "")

	if code := cmd.Seed(ctx, []string{"--profile", "dev"}); code != exit.OK {
		t.Fatalf("seeding: code %d — %s", code, captured.err.String())
	}

	lines := strings.Split(strings.TrimSpace(captured.out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("the dev profile creates 3 accounts, got %d line(s): %q", len(lines), captured.out.String())
	}
	for _, line := range lines {
		if fields := strings.Split(line, "\t"); len(fields) != 3 || fields[2] != "pending" {
			t.Fatalf("a seeded account must be identical to a registered one: %q", line)
		}
	}

	// Replaying the same profile must CONFLICT: the proof that the first pass
	// really did write into the store, through the use case.
	if code := cmd.Seed(ctx, []string{"--profile", "dev"}); code != exit.DataErr {
		t.Fatalf("a second seeding must conflict, got %d", code)
	}
}
