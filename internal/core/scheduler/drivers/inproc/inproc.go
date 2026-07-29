// Package inproc implements the election inside a single process.
//
// # NON-GUARANTEES — to be read before using it
//
//   - **No election between replicas.** This is THE non-guarantee, and it
//     cancels out the reason the module exists: two instances will run the task
//     twice. A reminder sent twice is visible to the customer; an invoice issued
//     twice is visible in the accounts.
//   - **No persistence.** A restart during an execution leaves the task
//     unfinished, with no trace.
//
// What it does guarantee, on the other hand, and it is not nothing: **no
// overlap**. A task still in progress when the next tick arrives is not
// relaunched. Without that, a task slower than its period would pile executions
// up until the process was saturated.
//
// Suitable in development, in test, for a CLI and for any deployment with ONE
// single replica. NOT suitable as soon as there are two — move then to the
// `advisory-lock` driver, without touching the calling code.
package inproc

import (
	"context"
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Elector grants the execution within the current process.
type Elector struct {
	mu      sync.Mutex
	running map[domain.TaskName]struct{}
}

// New builds the elector.
func New() *Elector {
	return &Elector{running: make(map[domain.TaskName]struct{})}
}

// Acquire implements ports.Acquire.
//
// Returns `false` if the task is already in progress in this process. The lock
// is held throughout the decision: testing it then setting it in two steps
// would let two goroutines obtain it.
func (e *Elector) Acquire(_ context.Context, task domain.TaskName) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, busy := e.running[task]; busy {
		return false, nil
	}
	e.running[task] = struct{}{}
	return true, nil
}

// Release implements ports.Release.
func (e *Elector) Release(_ context.Context, task domain.TaskName) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.running, task)
	return nil
}
