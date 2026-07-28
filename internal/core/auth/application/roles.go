package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewDefineRole compose la définition d'un rôle.
//
// Les permissions sont validées UNE par UNE, et la première fautive fait échouer
// l'ensemble. Un rôle à moitié défini serait pire qu'un rôle refusé : il
// accorderait quelque chose, sans que personne sache quoi.
func NewDefineRole(deps Deps) ports.DefineRole {
	return func(ctx context.Context, name string, raw []string) error {
		permissions := make([]domain.Permission, 0, len(raw))
		for _, candidate := range raw {
			permission, err := domain.NewPermission(candidate)
			if err != nil {
				return fmt.Errorf("rôle %q: %w", name, err)
			}
			permissions = append(permissions, permission)
		}

		role, err := domain.NewRole(name, permissions)
		if err != nil {
			return fmt.Errorf("définition du rôle: %w", err)
		}
		if err := deps.SaveRole(ctx, role); err != nil {
			return fmt.Errorf("enregistrement du rôle: %w", err)
		}
		return nil
	}
}

// NewAssignRoles compose l'affectation de rôles à une identité.
//
// Aucun contrôle d'existence des rôles ici, et c'est délibéré : affecter un rôle
// qui n'existe pas encore n'accorde RIEN — `Grants` ne trouvera pas de
// permissions — mais permet de provisionner dans l'ordre qu'on veut. Le refus
// serait une contrainte d'ordonnancement déguisée en règle de sécurité.
func NewAssignRoles(deps Deps) ports.AssignRoles {
	return func(ctx context.Context, id domain.IdentityID, roles []string) error {
		if id == "" {
			return fmt.Errorf("%w: l'identité est obligatoire", domain.ErrIncomplete)
		}
		if err := deps.BindRoles(ctx, id, roles); err != nil {
			return fmt.Errorf("affectation des rôles: %w", err)
		}
		return nil
	}
}
