package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewDefineRole composes the definition of a role.
//
// The permissions are validated ONE by ONE, and the first faulty one fails the
// whole thing. A half-defined role would be worse than a refused role: it would
// grant something, without anyone knowing what.
func NewDefineRole(deps Deps) ports.DefineRole {
	return func(ctx context.Context, name string, raw []string) error {
		permissions := make([]domain.Permission, 0, len(raw))
		for _, candidate := range raw {
			permission, err := domain.NewPermission(candidate)
			if err != nil {
				return fmt.Errorf("role %q: %w", name, err)
			}
			permissions = append(permissions, permission)
		}

		role, err := domain.NewRole(name, permissions)
		if err != nil {
			return fmt.Errorf("defining the role: %w", err)
		}
		if err := deps.SaveRole(ctx, role); err != nil {
			return fmt.Errorf("saving the role: %w", err)
		}
		return nil
	}
}

// NewAssignRoles composes the assignment of roles to an identity.
//
// No existence check on the roles here, and that is deliberate: assigning a
// role that does not exist yet grants NOTHING — `Grants` will find no
// permissions — but it allows provisioning in whatever order you like. Refusing
// would be a sequencing constraint dressed up as a security rule.
func NewAssignRoles(deps Deps) ports.AssignRoles {
	return func(ctx context.Context, id domain.IdentityID, roles []string) error {
		if id == "" {
			return fmt.Errorf("%w: the identity is mandatory", domain.ErrIncomplete)
		}
		if err := deps.BindRoles(ctx, id, roles); err != nil {
			return fmt.Errorf("assigning the roles: %w", err)
		}
		return nil
	}
}
