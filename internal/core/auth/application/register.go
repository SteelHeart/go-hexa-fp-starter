package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// secretMinLen bounds the length of a secret.
//
// Twelve characters, and no composition requirement — no uppercase, no digit,
// no special character. Length is the only constraint that really increases
// entropy; composition rules mostly push people to write `Password1!` and stick
// it under the keyboard.
const secretMinLen = 12

// NewRegister composes the creation of an identity and of its secret.
//
// # The order of the steps is the decision
//
// Validate the shape, hash, write. Hashing BEFORE writing is not a detail: it
// is what guarantees the plain secret never crosses the store's boundary. A
// driver only ever receives a digest, so it cannot log it even by accident.
//
// Subject uniqueness is decided by the STORE, not here: between a check and a
// write, two simultaneous requests both get through.
func NewRegister(deps Deps) ports.Register {
	return func(ctx context.Context, rawSubject, secret string) (domain.Identity, error) {
		subject, err := domain.NewSubject(rawSubject)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("subject: %w", err)
		}
		if len([]rune(secret)) < secretMinLen {
			return domain.Identity{}, fmt.Errorf(
				"%w: the secret must be at least %d characters long", domain.ErrIncomplete, secretMinLen)
		}

		hash, err := deps.HashSecret(secret)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("hashing the secret: %w", err)
		}

		identity, err := domain.NewIdentity(deps.NewIdentityID(), subject, nil, deps.Now())
		if err != nil {
			return domain.Identity{}, fmt.Errorf("creating the identity: %w", err)
		}

		credential, err := domain.NewCredential(identity, hash)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("assembling the credential: %w", err)
		}
		if err := deps.SaveIdentity(ctx, credential); err != nil {
			return domain.Identity{}, fmt.Errorf("saving the identity: %w", err)
		}
		return identity, nil
	}
}
