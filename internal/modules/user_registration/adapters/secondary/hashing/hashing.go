// Package hashing wires the HashPassword port onto the security infrastructure.
//
// Hashing is NOT domain: it is a costly, parameterised effect, whose parameters
// change with the hardware. The domain therefore only knows an opaque
// `PasswordHash`, and does not even know the name of the algorithm.
//
// Practical consequence: moving from Argon2id to whatever succeeds it does not
// touch a line of `domain/` or `application/`.
package hashing

import (
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// New builds the hashing port.
//
// Error contract: CodeInternal ONLY. A hashing failure is a technical defect —
// entropy unavailable, absurd parameters — never an error attributable to the
// user. Translating it into an input error would blame a password that is
// nonetheless valid.
func New(hasher security.Hasher) ports.HashPassword {
	return func(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
		// Expose() and NOT String(): String() returns "[redacted]", by design,
		// so that no accidental logging leaks the password. Using it here would
		// hash the SAME string for every account — they would all share one
		// digest, and any password would open any account.
		//
		// This is the only place in the repository allowed to call Expose(),
		// and the name is explicit so that this call is seen in review.
		encoded, err := hasher.Hash(password.Expose())
		if err != nil {
			// The returned message names neither the algorithm nor the cause:
			// both tell an attacker something about what runs behind.
			return result.Err[domain.PasswordHash, domain.Error](
				domain.NewError(
					domain.CodeInternal,
					"le mot de passe n'a pas pu être traité",
				).WithCause(err),
			)
		}
		return result.Ok[domain.PasswordHash, domain.Error](domain.NewPasswordHash(encoded))
	}
}
