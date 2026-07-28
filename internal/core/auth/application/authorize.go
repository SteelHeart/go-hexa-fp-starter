package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewAuthorize compose la vérification d'une permission.
//
// # C'est la fonction qui porte toute l'ADR 017
//
// Elle interroge `Grants`, c'est-à-dire l'état PERSISTÉ, à chaque appel. Elle ne
// lit aucune revendication portée par un jeton, et ne met rien en cache.
//
// Trois lignes, et c'est le point : il n'y a nulle part où glisser une
// optimisation qui rendrait un droit révoqué encore actif. Le jour où le coût de
// cet appel deviendra un goulot, la réponse sera un décorateur de cache EXPLICITE
// et borné sur ce port — pas des permissions dans le jeton.
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
