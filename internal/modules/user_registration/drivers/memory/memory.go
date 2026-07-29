// Package memory implements registration persistence in memory.
//
// # Why this driver exists
//
// It is not a test stub: it is the module's DEFAULT driver, and it is what makes
// the starter's promise true — `hexa new` then `go run` starts with no database,
// no Docker, nothing at all. A business module whose only driver required
// PostgreSQL would break that promise on the first module written, that is to
// say at the exact moment one seeks to exercise it.
//
// # GUARANTEES
//
//   - **Address uniqueness**: the map is indexed by normalised address, and
//     `SaveUser` refuses a duplicate with `CodeEmailAlreadyExists` — the same
//     code the SQL driver will return on a constraint violation. That is what
//     makes both drivers genuinely substitutable (ADR 003).
//   - **Concurrency safe**: every read and write goes through an RWMutex.
//   - **No mutation of the input**: `domain.User` is a value, it is copied on
//     the way in as on the way out.
//
// # NON-GUARANTEES
//
//   - **No durability.** Everything disappears when the process stops. That is
//     the expected behaviour of a memory driver, and it must be chosen knowingly
//     — never endured.
//   - **No sharing between replicas.** Two instances have two stores, therefore
//     two truths. Address uniqueness is guaranteed ONLY within one process.
//   - **No transaction.** The module's `RunInTx` boils down to a direct call: a
//     failure after the write does not cancel it. This is the costliest
//     NON-GUARANTEE, and the reason why this driver is unfit for production.
//   - **Memory grows without end.** No purge, no bound.
package memory

import (
	"context"
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Store retains the registered users, indexed by normalised address.
//
// The index is the address and not the identifier: every operation of the module
// queries by address. Indexing by identifier would force a walk over the map on
// every uniqueness check — and above all, would make address uniqueness depend
// on call discipline rather than on the structure.
type Store struct {
	mu      sync.RWMutex
	byEmail map[domain.Email]domain.User
}

// New builds an empty store.
func New() *Store {
	return &Store{byEmail: make(map[domain.Email]domain.User)}
}

// Save implements ports.SaveUser.
//
// The duplicate refusal is here and not in the use case, deliberately. The use
// case already checks availability, but between its check and its write there is
// a window: two simultaneous registrations on the same address both cross it.
// Only the store, which holds the lock, can settle it — exactly as the SQL
// uniqueness constraint settles it for the postgres driver.
func (s *Store) Save(_ context.Context, user domain.User) result.Result[domain.User, domain.Error] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return result.Err[domain.User, domain.Error](domain.NewError(
			domain.CodeEmailAlreadyExists,
			"cette adresse de courriel est déjà enregistrée",
		).WithField("email"))
	}

	s.byEmail[user.Email] = user
	return result.Ok[domain.User, domain.Error](user)
}

// IsTaken implements ports.EmailIsTaken.
//
// An absence is NOT an error: it amounts to `false`. That is the port's
// contract.
func (s *Store) IsTaken(_ context.Context, email domain.Email) result.Result[bool, domain.Error] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.byEmail[email]
	return result.Ok[bool, domain.Error](exists)
}

// FindByEmail implements ports.FindUserByEmail.
//
// An absence returns `None`, never an error: the Option makes absence explicit
// in the type, therefore impossible to confuse with a breakdown.
func (s *Store) FindByEmail(
	_ context.Context,
	email domain.Email,
) result.Result[fp.Option[domain.User], domain.Error] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byEmail[email]
	if !exists {
		return result.Ok[fp.Option[domain.User], domain.Error](fp.None[domain.User]())
	}
	return result.Ok[fp.Option[domain.User], domain.Error](fp.Some(user))
}

// Count returns the number of retained users.
//
// Exposed for operations and probes, not for tests: those go through the ports,
// like any other caller.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byEmail)
}
