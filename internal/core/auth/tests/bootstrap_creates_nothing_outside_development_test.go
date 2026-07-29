package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestBootstrapCreatesNothingOutsideDevelopment is the guard that makes
// bootstrapping acceptable.
//
// # What it prevents
//
// A demo account created in production. The shortcut is tolerable ONLY because
// it does not exist anywhere but locally — and "it does not exist anywhere
// else" is a claim that is either checked or degrades.
//
// The test records both halves: nothing is returned in the report, AND no
// identity really exists — because an empty report proves nothing if the write
// happened anyway.
func TestBootstrapCreatesNothingOutsideDevelopment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	for _, env := range []config.Environment{config.EnvProduction, config.EnvUAT, "anything at all", ""} {
		mod, _ := newModule(t, nil)

		report, err := auth.Bootstrap(ctx, mod, env)
		if err != nil {
			t.Fatalf("env %q: bootstrapping refuses to act, it does not fail — %v", env, err)
		}
		if report.Created || report.Subject != "" || report.Secret != "" {
			t.Fatalf("env %q: bootstrapping acted outside development — %+v", env, report)
		}

		// The proof that matters: no account exists. An empty report can be
		// faked in one line; a populated store cannot.
		_, err = mod.Authenticate(ctx, auth.BootstrapSubject, "any secret whatsoever")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("env %q: a bootstrap account exists — %v", env, err)
		}
	}
}

// TestBootstrapOpensASessionInDevelopment closes the time to first success.
//
// This is the acceptance criterion of #99: a freshly started server must make
// it possible to obtain a session. Before, `POST /v1/auth/sessions` returned
// 401 to everybody, without exception, for want of a creatable account.
//
// The secret is GENERATED, so the test cannot know it in advance — it reads it
// from the report, which is exactly what the operator will do reading their
// log.
func TestBootstrapOpensASessionInDevelopment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	report, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("bootstrapping: %v", err)
	}
	if !report.Created {
		t.Fatal("bootstrapping should have created an account in development")
	}
	if len(report.Secret) < 12 {
		t.Fatalf("the generated secret must satisfy the module's bounds, got %d characters", len(report.Secret))
	}

	if _, err := mod.Authenticate(ctx, report.Subject, report.Secret); err != nil {
		t.Fatalf("the bootstrapped account must authenticate: %v", err)
	}
}

// TestBootstrapEngendersADistinctSecretEachTime guards the randomness.
//
// A secret derived from the subject, from the clock or from a constant would be
// guessable — and a development workstation is often reachable from the local
// network. The fact that a secret is "only" for development does not make it
// any less usable by someone else.
func TestBootstrapEngendersADistinctSecretEachTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	seen := make(map[string]bool)

	for range 5 {
		mod, _ := newModule(t, nil)
		report, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
		if err != nil {
			t.Fatalf("bootstrapping: %v", err)
		}
		if seen[report.Secret] {
			t.Fatalf("secret reused from one bootstrap to the next: %q", report.Secret)
		}
		seen[report.Secret] = true
	}
}

// TestBootstrapIsIdempotentAndNeverResetsAnAccount guards the existing account.
//
// Restarting a server must not reset an existing account, nor return a secret
// nobody knows. The second call creates nothing and claims nothing: that is
// what prevents an operator from believing they received the current password
// when they are reading that of an account that was never created.
func TestBootstrapIsIdempotentAndNeverResetsAnAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	first, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}

	second, err := auth.Bootstrap(ctx, mod, config.EnvDevelopment)
	if err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	if second.Created || second.Secret != "" {
		t.Fatalf("the second bootstrap acted: %+v", second)
	}

	// The first secret is STILL worth something: nothing was reset.
	if _, err := mod.Authenticate(ctx, first.Subject, first.Secret); err != nil {
		t.Fatalf("the account was reset by the second bootstrap: %v", err)
	}
}

// TestBootstrapNeverBreaksTheStartupOfADisabledModule guards startup.
//
// # This test said EXACTLY the opposite, and it was wrong
//
// It required the refusal of a turned-off module to surface, on the grounds
// that a startup announcing "account bootstrapped" on a disabled module would
// send people looking for the fault on the password side. The reasoning was
// about the wrong risk: surfacing the error **fails the whole startup** as soon
// as `auth` is disabled.
//
// That happened. `IsLocal()` covers `development` AND `test`, whereas enabling
// only comes from the `development` layer: under `APP_ENV=test`, the server
// refused to start with `auth module disabled`. The end-to-end CI found it; the
// local measurement could not, it had covered `development` and `production` —
// never the one combination that breaks, **local environment AND module turned
// off**.
//
// A disabled module is not an outage, it is a configured state. There is
// nothing to bootstrap, and saying so through a fatal error is a fail-closed
// put in the wrong place.
func TestBootstrapNeverBreaksTheStartupOfADisabledModule(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	disabled, err := auth.New(config.Module{Enabled: false}, auth.Deps{})
	if err != nil {
		t.Fatalf("mounting the disabled module: %v", err)
	}

	for _, env := range []config.Environment{config.EnvDevelopment, config.EnvTest} {
		report, err := auth.Bootstrap(ctx, disabled, env)
		if err != nil {
			t.Fatalf("env %q: a turned-off module must not fail startup — %v", env, err)
		}
		if report.Created || report.Secret != "" {
			t.Fatalf("env %q: a disabled module creates nothing — %+v", env, report)
		}
	}
}
