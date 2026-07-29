package outbox

import (
	"errors"
	"fmt"
)

// ErrNotShared refuses a driver that does not cross the process boundary.
//
// # The defect this refusal prevents
//
// The `memory` driver lives INSIDE the process. A dispatcher launched as a
// separate binary would therefore query its own store — empty — while the
// events written by the server would stay in the server's memory.
//
// It would run publishing nothing AND without any error: the logs would be
// clean, the probe green, the process alive. The defect would only be
// discovered the day someone asks why a customer never received their email —
// and one would then have to walk back the whole chain to find that the link
// had been running empty since day one.
//
// A silently inert component is the worst possible defect: it is the only one
// that never signals itself.
var ErrNotShared = errors.New("outbox driver not shared across processes")

// SharedAcrossProcesses states whether a driver is visible from a process
// OTHER than the one that writes.
//
// This knowledge lives here, in the module, and not in the caller: the module
// is the only one that knows what each of its drivers guarantees. A new driver
// therefore adds its answer here, in the same place as its wiring — rather than
// in a list kept up to date elsewhere, which would end up diverging.
//
// Deny by default: an unknown driver is deemed NOT shared. Erring in that
// direction fails a startup; erring in the other makes a dispatcher run empty.
func SharedAcrossProcesses(driver string) bool {
	switch driver {
	case driverPostgres:
		return true
	case driverMemory:
		return false
	default:
		return false
	}
}

// RequireSharedDriver refuses a configuration on which a SEPARATE dispatcher
// would never see anything.
//
// To be called by any dispatching binary, never by the server: the latter
// writes to the outbox and therefore has no reason to require a shared driver.
func RequireSharedDriver(driver string) error {
	if SharedAcrossProcesses(driver) {
		return nil
	}
	return fmt.Errorf(
		"%w: %q. A separately launched dispatcher would never see the events "+
			"written by the server. Switch modules.%s.driver to a shared driver",
		ErrNotShared, driver, Name)
}
