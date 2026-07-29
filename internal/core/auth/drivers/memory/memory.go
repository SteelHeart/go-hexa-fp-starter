// Package memory implements the authentication store in memory.
//
// # Why this driver exists
//
// It is not a test stub: it is the DEFAULT driver, and it is what makes the
// starter's promise true — `hexa new` then `go run` starts without a database,
// without Docker, **with authentication enabled**. An `auth` module whose only
// driver required PostgreSQL would break that promise at the exact moment one
// sets out to demonstrate it.
//
// # File map
//
//	memory.go       the store itself, and its guarantees
//	identities.go   identities and digests
//	sessions.go     issued tokens, revocation, purge
//	roles.go        roles, assignment, and the "does it hold this right?" question
//
// # GUARANTEES
//
//   - **Subject uniqueness**: indexed by normalised subject, `SaveIdentity`
//     refuses a duplicate with `ErrSubjectTaken` — the same code an SQL driver
//     will return on a constraint violation. That is what makes two drivers
//     really substitutable.
//   - **Immediate revocation**: a deleted session stops being worth anything on
//     the next call, a withdrawn role stops granting on the next call, and a
//     closed account stops being worth anything everywhere on the next call.
//     That is decision 1 of ADR 017, upheld by the store.
//   - **Concurrency safe**: every read and write goes through an RWMutex.
//
// # NON-GUARANTEES
//
//   - **No durability.** Everything disappears on shutdown: restarting signs
//     everyone out and ERASES THE ACCOUNTS. That is acceptable in development
//     and unacceptable anywhere else.
//   - **No sharing between replicas.** Two instances have two stores: a session
//     issued by one is unknown to the other. Behind a load balancer, one request
//     out of two would fail with a 401.
//   - **No automatic purge.** Expired sessions stay in memory until an explicit
//     call to `PurgeExpired`; they are no longer worth anything, but they take
//     up room.
package memory

import (
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// Store keeps identities, secrets, roles and sessions.
//
// A SINGLE lock for the four maps, deliberately. They are read together —
// `Grants` walks the credentials AND the roles — and separate locks would gain
// nothing but an acquisition order to respect, hence a deadlock to discover in
// production.
type Store struct {
	mu          sync.RWMutex
	bySubject   map[string]domain.IdentityID
	credentials map[domain.IdentityID]domain.Credential
	roles       map[string]domain.Role
	sessions    map[string]domain.Session
}

// New builds an empty store.
func New() *Store {
	return &Store{
		bySubject:   make(map[string]domain.IdentityID),
		credentials: make(map[domain.IdentityID]domain.Credential),
		roles:       make(map[string]domain.Role),
		sessions:    make(map[string]domain.Session),
	}
}
