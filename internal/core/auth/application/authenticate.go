package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewAuthenticate composes the exchange of a secret for a session.
//
// # The order of the steps is the decision, and it reads
//
// Validate the shape, look up the credential, compare the secret, issue the
// session. An inversion would issue before comparing — that is, authenticate
// anybody.
//
// # A single refusal for three causes, deliberately
//
// Malformed subject, unknown subject, wrong secret: all return
// `domain.ErrInvalidCredentials`. Distinguishing "this account does not exist"
// from "the password is wrong" turns the sign-in form into an **account
// existence oracle** — you query a thousand addresses, note which ones answer
// differently, and you know where to concentrate the effort.
//
// ⚠️ This function nevertheless remains an oracle through TIME: an unknown
// subject answers without any hashing, hence faster. The remedy is to hash
// anyway, and it is not applied here — see the NON-guarantee written at the end
// of this file.
func NewAuthenticate(deps Deps) ports.Authenticate {
	return func(ctx context.Context, rawSubject, secret string) (domain.Session, error) {
		subject, err := domain.NewSubject(rawSubject)
		if err != nil {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		credential, err := deps.FindBySubject(ctx, subject)
		if err != nil {
			return domain.Session{}, unmasked(err)
		}
		if !credential.Identity().Active {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		matches, err := deps.VerifySecret(secret, credential.SecretHash())
		if err != nil {
			return domain.Session{}, fmt.Errorf("comparing the secret: %w", err)
		}
		if !matches {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		return deps.issue(ctx, credential.Identity().ID)
	}
}

// unmasked returns the refusal BARE when it is a credentials refusal.
//
// # The defect this function fixes
//
// Wrapping the sentinel — `fmt.Errorf("authentication: %w", err)` — reopens the
// oracle through the MESSAGE. `errors.Is` still recognises the refusal, so
// every taxonomy test stays green; but `err.Error()` returns "authentication:
// invalid credentials" for an unknown subject and "invalid credentials" for a
// wrong secret. A surface that logs or returns the message then tells the two
// cases apart — exactly what the single message was trying to hide,
// reintroduced by a wrapper added out of reflex.
//
// Any OTHER error stays wrapped: a store outage is not a credentials refusal,
// and conflating them would tell someone their password is wrong when the
// database is down.
func unmasked(err error) error {
	if errors.Is(err, domain.ErrInvalidCredentials) {
		return domain.ErrInvalidCredentials
	}
	return fmt.Errorf("authentication: %w", err)
}

// issue produces and persists a session.
//
// Isolated so that `NewAuthenticate` stays under `arch-go`'s line threshold,
// and because it names the only place in the module where a token is born.
func (d Deps) issue(ctx context.Context, id domain.IdentityID) (domain.Session, error) {
	token, err := d.NewToken()
	if err != nil {
		return domain.Session{}, fmt.Errorf("producing the token: %w", err)
	}

	session, err := domain.NewSession(token, id, d.Now(), d.SessionTTL)
	if err != nil {
		return domain.Session{}, fmt.Errorf("opening the session: %w", err)
	}
	if err := d.SaveSession(ctx, session); err != nil {
		return domain.Session{}, fmt.Errorf("saving the session: %w", err)
	}
	return session, nil
}

// NON-GUARANTEE — the timing oracle is not closed.
//
// An unknown subject returns `ErrInvalidCredentials` WITHOUT having hashed
// anything; a known subject pays for a full Argon2id, that is several tens of
// milliseconds. The difference is measurable on a local network, and it reveals
// which accounts exist — exactly what the single message tries to hide.
//
// The remedy is known: hash against a dummy digest when the subject is unknown,
// so that both paths cost the same. It is NOT applied here because it requires
// a reference digest produced with the same parameters as those configured, and
// building it at startup is an infrastructure decision, not a use case one.
//
// Written down rather than kept quiet: a named weakness gets fixed, an ignored
// weakness gets discovered in an audit.
