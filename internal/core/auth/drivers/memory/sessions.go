package memory

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveSession enregistre une session émise.
//
// Indexée par le JETON et non par l'identité : deux connexions du même compte —
// un téléphone et un poste — sont deux sessions. Indexer par l'identité ferait
// que se connecter quelque part déconnecte ailleurs, et le défaut ne se
// remarquerait que le jour où quelqu'un utilise deux appareils.
func (s *Store) SaveSession(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.Token.String()] = session
	return nil
}

// FindSession retrouve une session par son jeton.
//
// Ne filtre PAS les sessions expirées : c'est le cas d'usage qui vérifie
// l'expiration, avec l'horloge qu'on lui a donnée. Un pilote qui filtrerait
// laisserait croire que la vérification est faite en aval, et le premier pilote
// qui ne filtrerait pas rouvrirait la faille en silence.
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

// PurgeExpired retire les sessions périmées.
//
// Non appelée automatiquement : ce pilote n'a ni horloge ni tâche de fond, et lui
// en donner une le rendrait moins prévisible en test. C'est l'exploitant — ou
// l'ordonnanceur — qui décide quand nettoyer.
//
// ⚠️ La purge est de l'HYGIÈNE, pas de la sécurité. Une session expirée ne vaut
// déjà plus rien, purgée ou non : c'est le cas d'usage qui la refuse. Confondre
// les deux ferait croire que la protection dépend d'un nettoyage — et elle
// disparaîtrait le jour où ce nettoyage échouerait à moitié.
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
