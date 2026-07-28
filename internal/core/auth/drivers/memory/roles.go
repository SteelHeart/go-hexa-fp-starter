package memory

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveRole enregistre ou remplace un rôle et ses permissions.
//
// Le nom est celui que `domain.NewRole` a normalisé — c'est la même forme que
// l'affectation d'un rôle produit. Retenir ici un nom brut ferait diverger les
// deux chemins, et le rôle n'accorderait plus rien sans qu'aucune erreur ne le
// signale.
func (s *Store) SaveRole(_ context.Context, role domain.Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.roles[role.Name] = role
	return nil
}

// AssignRoles remplace les rôles d'une identité.
//
// REMPLACE plutôt qu'ajoute : « retirer un rôle » doit être aussi simple que
// « en donner un ». Une API qui n'offrirait que l'ajout ferait écrire le retrait
// à la main, donc mal.
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

// Grants indique si une identité détient une permission, À CET INSTANT.
//
// Le magasin est interrogé à chaque appel — c'est la décision 1 de l'ADR 017. Une
// permission retirée cesse d'accorder immédiatement, sans attendre l'expiration
// d'un quelconque jeton.
//
// Un compte fermé n'accorde rien, quels que soient ses rôles : la vérification de
// `Active` est ici et non chez l'appelant, pour qu'aucun chemin ne puisse
// l'oublier.
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

// rebind reconstruit une créance avec de nouveaux rôles.
//
// `NewCredential` ne peut pas échouer ici : la créance vient du magasin, donc son
// identité et son condensé sont déjà valides. Le repli rend la créance INCHANGÉE
// plutôt qu'une créance vide — se tromper dans le sens « le droit n'a pas changé »
// est le seul repli acceptable pour un contrôle d'accès.
func rebind(credential domain.Credential, roles []string) domain.Credential {
	rebound, err := domain.NewCredential(
		credential.Identity().WithRoles(roles), credential.SecretHash())
	if err != nil {
		return credential
	}
	return rebound
}
