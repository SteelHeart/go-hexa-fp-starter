package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Register registers an account and returns a `sysexits` exit code.
//
// # The SAME port as the HTTP surface
//
// `POST /v1/users` and this command both call `mod.Register`. The module does
// not know which one is calling it, and nothing had to be added to it for it to
// serve both. That is the starter's property no. 2, measured rather than stated.
//
// # The password is NOT a flag
//
// It is read from standard input. A `--password` shows up in `ps`, in the shell
// history, and in command audit logs — three places nobody thinks to purge. It
// is the most commonplace mistake of an administration tool, and it costs
// nothing to avoid.
func (c Command) Register(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	email := fs.String("email", "", "email address of the account to create")
	fs.Usage = func() {
		fmt.Fprintln(c.Err, "usage: hexa-cli register --email <address>")
		fmt.Fprintln(c.Err, "  the password is read from standard input")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return exit.Usage
	}
	if *email == "" {
		fmt.Fprintln(c.Err, "error: --email is mandatory")
		fs.Usage()
		return exit.Usage
	}

	secret, err := c.readSecret()
	if err != nil {
		fmt.Fprintf(c.Err, "error: %v\n", err)
		return exit.Usage
	}

	return c.registerAccount(ctx, *email, secret)
}

// registerAccount calls the use case and translates its outcome.
//
// Split out so that `Register` stays under `arch-go`'s line threshold, and
// because it names the boundary: above it arguments are read, below it nothing
// but domain is spoken.
func (c Command) registerAccount(ctx context.Context, email, secret string) int {
	value, failure, ok := c.Module.Register(ctx, domain.RegistrationCommand{
		Email:    email,
		Password: secret,
	}).Get()
	if !ok {
		// The message comes from the DOMAIN, not from here. It is the same
		// requirement as on the HTTP side: two phrasings of one and the same
		// rule diverge, and it is the wrong one that speaks to the user.
		fmt.Fprintf(c.Err, "error: %s\n", failure.Message)
		if failure.Field != "" {
			fmt.Fprintf(c.Err, "field: %s\n", failure.Field)
		}
		return codeFor(failure)
	}

	// On STANDARD output, and nothing other than the fact: a downstream script
	// reads this line. Adding any decoration to it would break it.
	fmt.Fprintf(c.Out, "%s\t%s\t%s\n", value.ID, value.Email, value.Status)
	return exit.OK
}
