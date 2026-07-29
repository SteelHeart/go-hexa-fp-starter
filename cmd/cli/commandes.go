package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	usercli "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/cli"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// errSeedInProduction refuses to seed a production environment.
//
// # Why an explicit refusal, and why it matters
//
// `seed` creates demonstration accounts. In production that means credentials
// known to everyone who has read the repository — hence a way in, opened by a
// command believed to be harmless because it is harmless everywhere else.
//
// The refusal is a `sysexits.NoPerm`: it is neither a failure nor an input
// error, it is a deliberate "no". The distinction matters to a script, which
// must neither retry nor raise an alert as it would on an incident.
var errSeedInProduction = errors.New(
	"seed refused outside development and test: demonstration accounts in production are a way in")

// commandRegister wires the registration surface.
func commandRegister(ctx context.Context, args []string) int {
	assembly, err := compose(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exit.Config
	}
	defer assembly.conn.Close()

	command := usercli.Command{Module: assembly.users, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	return command.Register(ctx, args)
}

// commandSeed seeds the starter THROUGH THE USE CASES.
//
// Never a direct `INSERT`: a dataset forged in SQL bypasses the domain rules,
// and produces accounts the code would never have let be born — unhashed
// passwords, unnormalised addresses, impossible statuses. The day a test
// relies on that, it validates a state production cannot reach.
func commandSeed(ctx context.Context, args []string) int {
	assembly, err := compose(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exit.Config
	}
	defer assembly.conn.Close()

	if err := allowSeed(assembly.cfg.App.Env); err != nil {
		fmt.Fprintf(os.Stderr, "refused: %v\n", err)
		return exit.NoPerm
	}

	command := usercli.Command{Module: assembly.users, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	return command.Seed(ctx, args)
}

// allowSeed refuses seeding outside the local environments.
//
// The decision is taken on the RESOLVED `app.env`, not on a variable read
// separately: it is the same value as the one governing network hardening and
// the authentication bootstrap. Two sources for one question end up
// contradicting each other, and it is the permissive one that wins.
func allowSeed(env config.Environment) error {
	if !env.IsLocal() {
		return fmt.Errorf("%w (env=%s)", errSeedInProduction, env)
	}
	return nil
}

// commandHealth checks that this binary COULD start.
//
// # What it really checks
//
// That the catalogue assembles, that the configuration loads and validates,
// that the requested connections open, and that the modules mount. That is
// exactly what a start-up does before listening — so a green `health` means
// "this deployment would start".
//
// It does NOT replace `/readyz`: the latter queries a running service, the
// former queries a configuration. The two answer different questions, and
// conflating them would make people believe a green `health` proves a server
// answers.
func commandHealth(ctx context.Context) int {
	assembly, err := compose(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exit.Config
	}
	defer assembly.conn.Close()

	fmt.Fprintf(os.Stdout, "ok\tenv=%s\tbase=%t\tcache=%t\n",
		assembly.cfg.App.Env, assembly.conn.Pool != nil, assembly.conn.Cache != nil)
	return exit.OK
}
