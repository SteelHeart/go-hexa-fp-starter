package memory

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveIdentity enregistre une identité et le condensé de son secret.
//
// Le refus du doublon est ICI et non dans le cas d'usage, délibérément. Le cas
// d'usage vérifie déjà, mais entre sa vérification et son écriture il existe une
// fenêtre : deux inscriptions simultanées sur le même sujet la franchissent
// toutes les deux. Seul le magasin, qui détient le verrou, peut trancher —
// exactement comme une contrainte d'unicité SQL.
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

// FindBySubject retrouve une identité et son condensé.
//
// Un sujet inconnu rend `ErrInvalidCredentials`, pas une erreur « introuvable » :
// le cas d'usage doit être incapable de distinguer les deux, sinon il finirait
// par le dire au client.
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

// FindIdentity retrouve une identité par son identifiant.
func (s *Store) FindIdentity(_ context.Context, id domain.IdentityID) (domain.Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, known := s.credentials[id]
	if !known {
		return domain.Identity{}, domain.ErrTokenUnknown
	}
	return credential.Identity(), nil
}

// UpdateIdentity remplace l'identité retenue, en CONSERVANT le condensé.
//
// Le condensé n'est pas un paramètre : il est relu de la créance existante. Une
// signature qui le reprendrait obligerait l'appelant à le transporter — donc à le
// lire, donc à pouvoir le journaliser — pour fermer un compte, ce qui n'a aucune
// raison de faire circuler un secret.
func (s *Store) UpdateIdentity(_ context.Context, identity domain.Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	credential, known := s.credentials[identity.ID]
	if !known {
		return domain.ErrInvalidCredentials
	}
	updated, err := domain.NewCredential(identity, credential.SecretHash())
	if err != nil {
		return fmt.Errorf("mise à jour de l'identité: %w", err)
	}
	s.credentials[identity.ID] = updated
	return nil
}

// Count rend le nombre d'identités retenues.
//
// Exposé pour l'exploitation et les sondes, pas pour les tests : ceux-ci passent
// par les ports, comme n'importe quel appelant.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.credentials)
}
