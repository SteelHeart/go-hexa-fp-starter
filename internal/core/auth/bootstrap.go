package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// BootstrapRole is the role granted to the bootstrap account.
//
// # Why bootstrapping grants rights, and not only an account
//
// An account with no role grants NOTHING — deny by default. Bootstrapping would
// therefore produce an administrator who gets a 403 on every administration
// route: the time to first success would stay infinite, merely moved one notch
// along.
//
// The permissions granted are those of the administration surface, and nothing
// more. In particular, this role presumes no BUSINESS right: an application
// that mounts this starter composes its own.
const BootstrapRole = "admin"

// BootstrapSubject designates the bootstrap account.
//
// Constant and not configurable: a parameterisable subject would invite writing
// the production administrator account into a versioned file, and that is
// exactly what we want to make impossible. Here, the name says what it is —
// local, disposable, worthless anywhere else.
const BootstrapSubject = "admin@local"

// bootstrapSecretBytes is the size of the generated secret's randomness.
//
// 24 bytes, that is 32 characters in base64 — well beyond the twelve required,
// and short enough to be copied from a log without a mistake.
const bootstrapSecretBytes = 24

// BootstrapReport says what bootstrapping did, without logging it itself.
//
// The module does not log (`application/` does not either): it reports. It is
// the caller that decides the level, the format and the destination — and that
// is what makes it possible to test bootstrapping without parsing logs.
type BootstrapReport struct {
	// Created says whether an account was created BY THIS CALL.
	Created bool

	// Subject is the bootstrap account, empty if nothing was done.
	Subject string

	// Secret is the GENERATED secret, to be displayed exactly once.
	//
	// ⚠️ It will never be readable again: only its digest is kept. The field is
	// empty as soon as `Created` is false.
	Secret string
}

// Bootstrap creates a bootstrap account — IN DEVELOPMENT ONLY.
//
// # The problem this solves
//
// The authentication surface publishes no administration operation: exposing
// them without protecting them would open account creation to anyone, and
// protecting them demands a first administrator. A fresh server therefore
// answered **401 to everybody**, without exception — the module's time to first
// success was infinite (#99).
//
// # What makes this shortcut acceptable, and it alone
//
//  1. **It only applies locally.** Outside `development` and `test`, the
//     function creates NOTHING and says so in its report. This is not an
//     error — failing the startup of a production because it refuses a demo
//     account would be absurd — it is a refusal to act.
//  2. **The secret is GENERATED, never written down.** No default password
//     exists in a versioned artefact. That is the mistake that really matters
//     here: a starter shipped with `admin/admin` is a starter that deploys
//     `admin/admin`, and nobody changes it before the incident.
//  3. **It is idempotent.** An already taken subject is not an error, and
//     nothing is recreated: restarting does not reset an existing account.
//  4. **It never fails a startup.** A disabled module is not an outage, it is a
//     configured state: there is nothing to bootstrap, and saying so through a
//     fatal error would prevent starting any local environment where
//     authentication is turned off.
//
// # Why the secret is returned rather than logged here
//
// Because a core module does not log. The caller receives it and decides — and
// that boundary is what guarantees a secret does not end up in an observability
// collector because a module thought it was being helpful.
func Bootstrap(ctx context.Context, mod Module, env config.Environment) (BootstrapReport, error) {
	if !env.IsLocal() {
		// Refusal to act, not an error. The empty report IS the answer.
		return BootstrapReport{}, nil
	}

	secret, err := randomSecret()
	if err != nil {
		return BootstrapReport{}, err
	}

	identity, err := mod.Register(ctx, BootstrapSubject, secret)
	if err != nil {
		if errors.Is(err, domain.ErrSubjectTaken) || errors.Is(err, ErrDisabled) {
			// Two distinct "nothing to do", and the same empty report:
			//
			//   ErrSubjectTaken — already bootstrapped. We do not recreate, and
			//   we do not return the secret of an account whose password we no
			//   longer know.
			//
			//   ErrDisabled — the module is TURNED OFF. That is a configured
			//   state, not an outage: there is simply nothing to bootstrap.
			//
			// ⚠️ The second case used to surface as a fatal error, and FAILED
			// STARTUP as soon as `auth` was disabled. It was, in `test` —
			// `IsLocal()` covers `development` AND `test`, whereas enabling
			// only comes from the `development` layer. Found by the end-to-end
			// CI, never locally: the local measurement had covered
			// `development` and `production`, hence never the one combination
			// that breaks, local environment AND module turned off.
			return BootstrapReport{}, nil
		}
		return BootstrapReport{}, fmt.Errorf("bootstrapping account %q: %w", BootstrapSubject, err)
	}

	if err := grantAdmin(ctx, mod, identity.ID); err != nil {
		return BootstrapReport{}, err
	}
	return BootstrapReport{Created: true, Subject: BootstrapSubject, Secret: secret}, nil
}

// grantAdmin defines the bootstrap role and grants it.
//
// # Order matters, and failure must be loud
//
// Define before assigning: the reverse would work — a role assigned before it
// exists grants nothing, then grants — but would leave a window during which
// the account announced as administrator can do nothing.
//
// A failure here SURFACES rather than being swallowed. Returning a "created"
// report on an account with no rights would produce the worst possible message:
// the operator reads a secret, signs in, and gets a 403 everywhere without
// knowing why.
func grantAdmin(ctx context.Context, mod Module, id domain.IdentityID) error {
	permissions := []string{
		contract.PermissionIdentityCreate,
		contract.PermissionIdentityRoles,
		contract.PermissionIdentityClose,
		contract.PermissionRoleWrite,
	}
	if err := mod.DefineRole(ctx, BootstrapRole, permissions); err != nil {
		return fmt.Errorf("defining role %q: %w", BootstrapRole, err)
	}
	if err := mod.AssignRoles(ctx, id, []string{BootstrapRole}); err != nil {
		return fmt.Errorf("assigning role %q: %w", BootstrapRole, err)
	}
	return nil
}

// randomSecret draws a secret from a cryptographically secure source.
//
// `crypto/rand` and not `math/rand`, for the same reason as tokens: a
// predictable secret is a bypassed authentication. The fact that it is "only"
// for development changes nothing — a development workstation is often
// reachable from the local network.
func randomSecret() (string, error) {
	raw := make([]byte, bootstrapSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("entropy unavailable: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
