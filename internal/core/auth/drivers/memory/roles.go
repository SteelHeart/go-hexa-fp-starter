package memory

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveRole records or replaces a role and its permissions.
//
// The name is the one `domain.NewRole` normalised — it is the same form that
// assigning a role produces. Keeping a raw name here would make the two paths
// diverge, and the role would stop granting anything without any error
// reporting it.
func (s *Store) SaveRole(_ context.Context, role domain.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roles[role.Name] = role
	return nil
}

// AssignRoles replaces an identity's roles.
//
// REPLACES rather than adds: "withdraw a role" must be as simple as "grant
// one". An API that offered only addition would make withdrawal be written by
// hand, hence badly.
func (s *Store) AssignRoles(_ context.Context, id domain.IdentityID, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, known := s.credentials[id]
	if !known {
		return domain.ErrInvalidCredentials
	}
	s.credentials[id] = rebind(credential, roles)
	return nil
}

// Grants reports whether an identity holds a permission, AT THIS INSTANT.
//
// The store is queried on every call — that is decision 1 of ADR 017. A
// withdrawn permission stops granting immediately, without waiting for any
// token to expire.
//
// A closed account grants nothing, whatever its roles: the `Active` check is
// here and not at the caller, so that no path can forget it.
func (s *Store) Grants(_ context.Context, id domain.IdentityID, permission domain.Permission) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, known := s.credentials[id]
	if !known || !credential.Identity().Active {
		return false
	}
	for _, name := range credential.Identity().Roles {
		if role, exists := s.roles[name]; exists && role.Grants(permission) {
			return true
		}
	}
	return false
}

// rebind rebuilds a credential with new roles.
//
// `NewCredential` cannot fail here: the credential comes from the store, so its
// identity and its digest are already valid. The fallback returns the credential
// UNCHANGED rather than an empty one — being wrong in the "the right has not
// changed" direction is the only acceptable fallback for an access control.
func rebind(credential domain.Credential, roles []string) domain.Credential {
	rebound, err := domain.NewCredential(
		credential.Identity().WithRoles(roles), credential.SecretHash())
	if err != nil {
		return credential
	}
	return rebound
}
