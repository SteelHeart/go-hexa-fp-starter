package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewAuthorize composes the checking of a permission.
//
// # This is the function that carries the whole of ADR 017
//
// It queries `Grants`, that is to say the PERSISTED state, on every call. It
// reads no claim carried by a token, and caches nothing.
//
// Three lines, and that is the point: there is nowhere to slip in an
// optimisation that would keep a revoked right active. The day the cost of this
// call becomes a bottleneck, the answer will be an EXPLICIT and bounded cache
// decorator on this port — not permissions inside the token.
func NewAuthorize(deps Deps) ports.Authorize {
	return func(ctx context.Context, identity domain.IdentityID, permission domain.Permission) error {
		if identity == "" || permission.IsZero() {
			return domain.ErrIncomplete
		}
		if !deps.Grants(ctx, identity, permission) {
			return domain.ErrForbidden
		}
		return nil
	}
}
