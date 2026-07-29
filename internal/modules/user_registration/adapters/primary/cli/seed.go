package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Known seeding profiles.
//
// A closed table rather than a free string: a misspelled profile must REFUSE,
// never fall back on the smallest one. Falling back would create two accounts
// where a thousand were expected, and the defect would only show up the moment a
// load test returned flattering numbers.
func profiles() map[string]int {
	return map[string]int{
		// dev: enough to click through an interface without waiting.
		"dev": 3,
		// demo: enough to show a paginated list that really paginates.
		"demo": 25,
	}
}

// seedSecret is the password of the generated accounts.
//
// # Why a constant is acceptable here, and only here
//
// Because `seed` REFUSES to run outside development and test — the guard is in
// the composition root, with its test. These accounts therefore only exist on
// machines where they protect nothing.
//
// It would be unacceptable anywhere else: a default secret in a versioned
// artefact is a deployed secret. That is the reason why the bootstrap of `auth`
// GENERATES its own (ADR 017 §6) — there, the account outlives the environment
// in which it was born.
//
//nolint:gosec // seeding password: `seed` REFUSES outside development and test
const seedSecret = "graine-de-developpement"

// Seed seeds the starter through the USE CASES, never through an `INSERT`.
//
// # Why going through the use cases is not negotiable
//
// A data set fabricated in SQL bypasses the rules of the domain. It produces
// accounts the code would never have let come into existence: unhashed
// passwords, unnormalised addresses, impossible statuses. The day a test relies
// on them, it validates a state production cannot reach — and it will stay green
// while hiding a real regression.
//
// The corollary has a cost: seeding through the use cases is SLOW, because every
// account pays an Argon2id hash. That is accepted — slowness is the price of
// representativeness, and a `perf` profile requiring a thousand accounts would
// call for a distinct path, not for a bypass of this one.
func (c Command) Seed(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	profile := fs.String("profile", "dev", "seeding profile: dev, demo")
	fs.Usage = func() {
		fmt.Fprintln(c.Err, "usage: hexa-cli seed --profile <dev|demo>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return exit.Usage
	}

	count, known := profiles()[*profile]
	if !known {
		fmt.Fprintf(c.Err, "error: unknown profile %q — expected dev or demo\n", *profile)
		return exit.Usage
	}

	return c.populate(ctx, *profile, count)
}

// populate creates the accounts of the profile and returns the first failure
// code.
//
// # Stopping on the first failure, rather than carrying on
//
// A half-done seeding is worse than a refused one: one does not know what
// exists. Stopping dead leaves an incomplete but KNOWN state — the report says
// how many were created — where carrying on would produce silent holes in the
// middle of the data set.
func (c Command) populate(ctx context.Context, profile string, count int) int {
	for i := 1; i <= count; i++ {
		email := fmt.Sprintf("%s-%02d@example.test", profile, i)
		if code := c.registerAccount(ctx, email, seedSecret); code != exit.OK {
			fmt.Fprintf(c.Err, "seeding interrupted after %d account(s) out of %d\n", i-1, count)
			return code
		}
	}
	fmt.Fprintf(c.Err, "profile %s: %d account(s) created\n", profile, count)
	return exit.OK
}
