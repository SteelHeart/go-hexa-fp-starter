package memory

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// SaveSession records an issued session.
//
// Indexed by the TOKEN and not by the identity: two sign-ins of the same
// account — a phone and a workstation — are two sessions. Indexing by identity
// would mean signing in somewhere signs you out elsewhere, and the defect would
// only be noticed the day someone uses two devices.
func (s *Store) SaveSession(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.Token.String()] = session
	return nil
}

// FindSession looks up a session by its token.
//
// Does NOT filter out expired sessions: it is the use case that checks expiry,
// with the clock it was given. A driver that filtered would suggest the check
// happens downstream, and the first driver that did not filter would silently
// reopen the hole.
func (s *Store) FindSession(_ context.Context, token domain.Token) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, known := s.sessions[token.String()]
	if !known {
		return domain.Session{}, domain.ErrTokenUnknown
	}
	return session, nil
}

// DeleteSession revokes a session.
//
// Deleting an unknown session is NOT an error: the operation is idempotent,
// because a client who signs out twice has done nothing wrong.
func (s *Store) DeleteSession(_ context.Context, token domain.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token.String())
	return nil
}

// PurgeExpired removes stale sessions.
//
// Not called automatically: this driver has neither a clock nor a background
// task, and giving it one would make it less predictable in tests. It is the
// operator — or the scheduler — that decides when to clean up.
//
// ⚠️ The purge is HOUSEKEEPING, not security. An expired session is already
// worth nothing, purged or not: it is the use case that refuses it. Conflating
// the two would make people believe the protection depends on a clean-up — and
// it would vanish the day that clean-up half failed.
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
