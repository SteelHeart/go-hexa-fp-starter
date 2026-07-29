// Package postgres implements the election by PostgreSQL advisory lock.
//
// # GUARANTEES
//
//   - **A single replica runs**, with no additional component to operate: no
//     Zookeeper, no etcd, no lease to renew.
//   - **Automatic release when a replica dies.** The lock is a SESSION one: if
//     the process disappears, the database closes the connection and the lock
//     drops. No expired lease to clean up by hand — that is what makes this
//     mechanism preferable to a `leader` row in a table.
//
// # NON-GUARANTEES
//
//   - **Does not survive a primary failover.** Advisory locks are not
//     replicated: during a switchover, two replicas may believe themselves
//     elected. A task must therefore stay idempotent even with this driver.
//   - **A key collision would serialise two distinct tasks** (see
//     domain.LockKey). Improbable over 63 bits, and the worst case goes in the
//     right direction.
//   - **A task already in progress in the process is not relaunched**, as with
//     the `inproc` driver: local overlap is refused before a connection is even
//     opened.
//
// # State
//
// Written, NEVER run against a database. No migration is necessary — advisory
// locks do not create a table — but nothing proves this code yet.
package postgres

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Elector grants the execution to a single replica.
type Elector struct {
	pool *pgxpool.Pool

	// held retains the connection that HOLDS the lock.
	//
	// Indispensable: a session advisory lock belongs to its connection. Taking
	// it on a connection handed straight back to the pool amounts to locking
	// nothing — the connection would serve somebody else, lock included.
	mu   sync.Mutex
	held map[domain.TaskName]*pgxpool.Conn
}

// New builds the elector.
func New(pool *pgxpool.Pool) *Elector {
	return &Elector{pool: pool, held: make(map[domain.TaskName]*pgxpool.Conn)}
}

// Acquire implements ports.Acquire.
func (e *Elector) Acquire(ctx context.Context, task domain.TaskName) (bool, error) {
	if e.busy(task) {
		return false, nil
	}

	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("no connection available for the election of %s: %w", task, err)
	}

	var elected bool
	err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, domain.LockKey(task)).Scan(&elected)
	if err != nil {
		conn.Release()
		return false, fmt.Errorf("taking the election lock of %s: %w", task, err)
	}
	if !elected {
		conn.Release()
		return false, nil
	}

	e.mu.Lock()
	e.held[task] = conn
	e.mu.Unlock()
	return true, nil
}

// busy says whether the current process already holds the task.
func (e *Elector) busy(task domain.TaskName) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, held := e.held[task]
	return held
}

// Release implements ports.Release.
//
// # Why the connection is DESTROYED when the unlock fails
//
// Handing back to the pool a connection that still holds a session lock would
// make that lock permanent: nobody would ever release it, and the task would no
// longer run until the process restarted. Closing the connection forces the
// database to release everything the session was holding. Losing a connection
// is the lesser evil.
func (e *Elector) Release(ctx context.Context, task domain.TaskName) error {
	e.mu.Lock()
	conn, held := e.held[task]
	delete(e.held, task)
	e.mu.Unlock()

	if !held {
		return nil
	}

	var released bool
	err := conn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, domain.LockKey(task)).Scan(&released)
	if err != nil || !released {
		if hijacked := conn.Hijack(); hijacked != nil {
			// Close error ignored KNOWINGLY: we are already on a failure path,
			// whose error is returned just below. Raising it instead would mask
			// the real cause with its symptom, and adding it would give the
			// caller two errors for a single incident. The aim is not to close
			// cleanly, it is to guarantee that the session dies — and a failed
			// close kills it just as well.
			_ = hijacked.Close(ctx)
		}
		if err != nil {
			return fmt.Errorf("releasing the election lock of %s: %w", task, err)
		}
		return fmt.Errorf("election lock of %s not held by this session", task)
	}

	conn.Release()
	return nil
}
