package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewDeactivate composes the closure of an account.
//
// # What the closure reaches, and why that is enough
//
// It deletes no session. It does not have to: `Verify` re-reads the identity on
// every call and refuses an inactive identity, and `Grants` refuses likewise.
// The account therefore stops being worth anything on the next call, including
// for tokens already in circulation — that is decision 1 of ADR 017, and it
// makes free what would otherwise cost a sweep of the store.
//
// Deleting the sessions on top would be housekeeping, not security. Conflating
// the two would make people believe housekeeping is what protects, and the
// protection would vanish the day housekeeping half failed.
func NewDeactivate(deps Deps) ports.Deactivate {
	return deps.setActive(domain.Identity.Deactivated)
}

// NewReactivate composes the reopening of an account.
func NewReactivate(deps Deps) ports.Reactivate {
	return deps.setActive(domain.Identity.Reactivated)
}

// setActive factors out the two transitions.
//
// The transformation is passed as a parameter rather than a boolean: the caller
// names what it does, and there is nowhere one could write "active = true"
// inadvertently.
//
// Both are IDEMPOTENT: applying the same transition twice gives the same state.
// Two administrators reacting to the same incident must not cancel each other
// out.
func (d Deps) setActive(transition func(domain.Identity) domain.Identity) func(
	context.Context, domain.IdentityID,
) error {
	return func(ctx context.Context, id domain.IdentityID) error {
		if id == "" {
			return fmt.Errorf("%w: the identity is mandatory", domain.ErrIncomplete)
		}

		identity, err := d.FindIdentity(ctx, id)
		if err != nil {
			return fmt.Errorf("identity: %w", err)
		}
		if err := d.UpdateIdentity(ctx, transition(identity)); err != nil {
			return fmt.Errorf("updating the identity: %w", err)
		}
		return nil
	}
}
