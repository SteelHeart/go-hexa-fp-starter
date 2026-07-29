package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewVerify composes the resolution of a token into an identity.
//
// # Three causes, a single refusal
//
// Unknown token, expired session, deactivated identity: everything returns
// `domain.ErrTokenUnknown`. Answering "expired" rather than "unknown" would
// confirm that such a token existed, hence that an account exists.
//
// # Expiry is checked HERE, not only at the store
//
// A driver may purge lazily — the in-memory one does not purge at all. Relying
// on it to return only valid sessions would make security depend on an
// implementation detail, and the first driver that purged differently would
// reopen the hole without anything reporting it.
func NewVerify(deps Deps) ports.Verify {
	return func(ctx context.Context, token domain.Token) (domain.Identity, error) {
		if token.IsZero() {
			return domain.Identity{}, domain.ErrTokenUnknown
		}

		session, err := deps.FindSession(ctx, token)
		if err != nil {
			return domain.Identity{}, domain.ErrTokenUnknown
		}
		if session.Expired(deps.Now()) {
			return domain.Identity{}, domain.ErrTokenUnknown
		}

		identity, err := deps.FindIdentity(ctx, session.Identity)
		if err != nil || !identity.Active {
			return domain.Identity{}, domain.ErrTokenUnknown
		}
		return identity, nil
	}
}

// NewRevoke composes the revocation of a session.
//
// Idempotent: revoking an already unknown token is not an error. A client who
// signs out twice has done nothing wrong, and failing the second call would
// produce an error nobody would know how to handle.
func NewRevoke(deps Deps) ports.Revoke {
	return func(ctx context.Context, token domain.Token) error {
		if token.IsZero() {
			return domain.ErrIncomplete
		}
		return deps.DeleteSession(ctx, token)
	}
}
