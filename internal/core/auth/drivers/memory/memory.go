// Package memory implémente le magasin d'authentification en mémoire.
//
// # Pourquoi ce pilote existe
//
// Il n'est pas un bouchon de test : c'est le pilote PAR DÉFAUT, et c'est lui qui
// rend vraie la promesse du socle — `hexa new` puis `go run` démarre sans base,
// sans Docker, **avec l'authentification active**. Un module `auth` dont le seul
// pilote exigerait PostgreSQL casserait cette promesse au moment exact où l'on
// cherche à la démontrer.
//
// # GARANTIES
//
//   - **Unicité du sujet** : indexé par sujet normalisé, `SaveIdentity` refuse un
//     doublon avec `ErrSubjectTaken` — le même code qu'un pilote SQL rendra sur
//     une violation de contrainte. C'est ce qui rend deux pilotes réellement
//     substituables.
//   - **Révocation immédiate** : une session supprimée cesse de valoir au prochain
//     appel, et un rôle retiré cesse d'accorder au prochain appel. C'est la
//     décision 1 de l'ADR 017, tenue par le magasin.
//   - **Sûr en concurrence** : toutes les lectures et écritures passent par un
//     RWMutex.
//
// # NON-GARANTIES
//
//   - **Aucune durabilité.** Tout disparaît à l'arrêt : redémarrer déconnecte tout
//     le monde et EFFACE LES COMPTES. C'est acceptable en développement et
//     inacceptable ailleurs.
//   - **Aucun partage entre répliques.** Deux instances ont deux magasins : une
//     session émise par l'une est inconnue de l'autre. Derrière un répartiteur de
//     charge, une requête sur deux échouerait en 401.
//   - **Aucune purge.** Les sessions expirées restent en mémoire jusqu'au
//     redémarrage ; elles ne valent plus rien, mais elles occupent.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// Store retient identités, secrets, rôles et sessions.
type Store struct {
	mu          sync.RWMutex
	bySubject   map[string]domain.IdentityID
	credentials map[domain.IdentityID]domain.Credential
	roles       map[string]domain.Role
	sessions    map[string]domain.Session
}

// New construit un magasin vide.
func New() *Store {
	return &Store{
		bySubject:   make(map[string]domain.IdentityID),
		credentials: make(map[domain.IdentityID]domain.Credential),
		roles:       make(map[string]domain.Role),
		sessions:    make(map[string]domain.Session),
	}
}

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

// SaveSession enregistre une session émise.
func (s *Store) SaveSession(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.Token.String()] = session
	return nil
}

// FindSession retrouve une session par son jeton.
func (s *Store) FindSession(_ context.Context, token domain.Token) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, known := s.sessions[token.String()]
	if !known {
		return domain.Session{}, domain.ErrTokenUnknown
	}
	return session, nil
}

// DeleteSession révoque une session.
//
// Supprimer une session inconnue n'est PAS une erreur : l'opération est
// idempotente, parce qu'un client qui se déconnecte deux fois n'a rien fait de mal.
func (s *Store) DeleteSession(_ context.Context, token domain.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token.String())
	return nil
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

// SaveRole enregistre ou remplace un rôle et ses permissions.
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
	s.credentials[id] = mustRebind(credential, roles)
	return nil
}

// Grants indique si une identité détient une permission, À CET INSTANT.
//
// Le magasin est interrogé à chaque appel — c'est la décision 1 de l'ADR 017. Une
// permission retirée cesse d'accorder immédiatement, sans attendre l'expiration
// d'un quelconque jeton.
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

// Count rend le nombre d'identités retenues.
//
// Exposé pour l'exploitation et les sondes, pas pour les tests : ceux-ci passent
// par les ports, comme n'importe quel appelant.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.credentials)
}

// mustRebind reconstruit une créance avec de nouveaux rôles.
//
// `NewCredential` ne peut pas échouer ici : la créance vient du magasin, donc son
// identité et son condensé sont déjà valides. L'erreur est ignorée en connaissance
// de cause plutôt que propagée dans une signature qu'elle ne concerne pas.
func mustRebind(credential domain.Credential, roles []string) domain.Credential {
	rebound, err := domain.NewCredential(
		credential.Identity().WithRoles(roles), credential.SecretHash())
	if err != nil {
		return credential
	}
	return rebound
}

// PurgeExpired retire les sessions périmées.
//
// Non appelée automatiquement : ce pilote n'a ni horloge ni tâche de fond, et lui
// en donner une le rendrait moins prévisible en test. C'est l'exploitant — ou
// l'ordonnanceur — qui décide quand nettoyer.
func (s *Store) PurgeExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for key, session := range s.sessions {
		if session.Expired(now) {
			delete(s.sessions, key)
			removed++
		}
	}
	return removed
}
