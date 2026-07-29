package outbox

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares this module's drivers — ADR 014.
//
// # Why here, and not in internal/config
//
// This table used to live in `internal/config/modules.go`, two packages away
// from the factory that actually builds the drivers. The comment that came with
// it already confessed to fearing divergence: « a typo in either of the two
// would make a module unactivatable, with a message that blames the user's
// configuration ».
//
// It is now in the SAME package as the `switch` of `New`, often on the same
// screen. Divergence does not become impossible — nothing checks it
// mechanically, and ADR 014 notes this as its weakness [humain] — but it
// becomes improbable.
//
// dispatcherOptions enumerates the keys read by `policyFrom`.
//
// A function rather than a package variable: `gochecknoglobals` refuses global
// state, and it is right — a shared slice is modifiable by any caller,
// including by accident.
func dispatcherOptions() []string {
	return []string{OptionBatchSize, OptionMaxAttempts, OptionBaseBackoff, OptionInterval}
}

// Guaranteed publication of events.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default requires NOTHING: this is what makes « `go run` starts »
			// true without a database, without a cache, without a container
			// (ADR 012).
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// Both drivers read the SAME dispatching policy: it describes the
				// behaviour of the dispatcher, not the storage.
				//
				// The keys are referenced, never rewritten. An admitted key that
				// nobody reads — or one read without being admitted — would be a
				// divergence between two lists; sharing the constant makes it
				// impossible (#93).
				//
				// Lost on restart: the process IS the storage.
				driverMemory: {Options: dispatcherOptions()},
				// Durable and transactional — the only form that keeps the promise of ADR 006.
				driverPostgres: {SQL: true, Options: dispatcherOptions()},
			},
		},
	}
}
