package memory

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveIdentity records an identity and the digest of its secret.
//
// The duplicate refusal is HERE and not in the use case, deliberately. The use
// case already checks, but between its check and its write there is a window:
// two simultaneous registrations on the same subject both cross it. Only the
// store, which holds the lock, can decide — exactly like an SQL uniqueness
// constraint.
func (s *Store) SaveIdentity(_ context.Context, credential domain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subject := credential.Identity().Subject.String()
	if _, exists := s.bySubject[subject]; exists {
		return domain.ErrSubjectTaken
	}
	s.bySubject[subject] = credential.Identity().ID
	s.credentials[credential.Identity().ID] = credential
	return nil
}

// FindBySubject looks up an identity and its digest.
//
// An unknown subject returns `ErrInvalidCredentials`, not a "not found" error:
// the use case must be unable to tell the two apart, otherwise it would end up
// saying so to the client.
func (s *Store) FindBySubject(_ context.Context, subject domain.Subject) (domain.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, known := s.bySubject[subject.String()]
	if !known {
		return domain.Credential{}, domain.ErrInvalidCredentials
	}
	credential, found := s.credentials[id]
	if !found {
		return domain.Credential{}, domain.ErrInvalidCredentials
	}
	return credential, nil
}

// FindIdentity looks up an identity by its identifier.
func (s *Store) FindIdentity(_ context.Context, id domain.IdentityID) (domain.Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, known := s.credentials[id]
	if !known {
		return domain.Identity{}, domain.ErrTokenUnknown
	}
	return credential.Identity(), nil
}

// UpdateIdentity replaces the stored identity, KEEPING the digest.
//
// The digest is not a parameter: it is re-read from the existing credential. A
// signature that took it back would force the caller to carry it — hence to
// read it, hence to be able to log it — in order to close an account, which has
// no reason to make a secret circulate.
func (s *Store) UpdateIdentity(_ context.Context, identity domain.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, known := s.credentials[identity.ID]
	if !known {
		return domain.ErrInvalidCredentials
	}
	updated, err := domain.NewCredential(identity, credential.SecretHash())
	if err != nil {
		return fmt.Errorf("updating the identity: %w", err)
	}
	s.credentials[identity.ID] = updated
	return nil
}

// Count returns the number of identities kept.
//
// Exposed for operations and probes, not for tests: those go through the ports,
// like any other caller.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.credentials)
}
