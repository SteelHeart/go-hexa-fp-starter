package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// bootstrapAuthentication creates the bootstrap account and ANNOUNCES it, once.
//
// # Why the secret is written to the log
//
// Because the alternative is worse. A default password in a versioned file is
// a deployed password: nobody changes it before the incident. A generated
// secret, shown a single time, and gone on restart cannot follow the project
// into production.
//
// The trade-off is BOUNDED: `auth.Bootstrap` creates nothing outside
// `development` and `test`, and the refusal has its test. Here, the log can
// therefore only contain a secret on a development machine.
//
// # Why it is not the module's job to log it
//
// A core module does not log — it reports. That boundary is what guarantees a
// secret does not leave for an observability collector because a module meant
// well. The decision to write is taken here, once, in a place people re-read.
func bootstrapAuthentication(
	ctx context.Context, mod auth.Module, env config.Environment, logger *slog.Logger,
) error {
	report, err := auth.Bootstrap(ctx, mod, env)
	if err != nil {
		return fmt.Errorf("authentication bootstrap: %w", err)
	}
	if !report.Created {
		return nil
	}

	logger.WarnContext(ctx,
		"BOOTSTRAP ACCOUNT CREATED — development only, shown a single time",
		slog.String("sujet", report.Subject),
		slog.String("secret", report.Secret),
		slog.String("note", "lost on restart: the memory driver keeps nothing"),
	)
	return nil
}
