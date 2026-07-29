package audit

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares the drivers of this module — ADR 014.
//
// # Why here, and not in internal/config
//
// This table used to live in `internal/config/modules.go`, two packages away
// from the factory that actually builds the drivers. The comment that came
// with it already admitted to fearing divergence: "a typo in either of the two
// would make a module impossible to activate, with a message that blames the
// user's configuration".
//
// It now lives in the SAME package as the `switch` of `New`, often on the same
// screen. Divergence does not become impossible — nothing checks it
// mechanically, and ADR 014 notes this as its weakness [human] — but it
// becomes improbable.
//
// Log of deeds, timestamped.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default requires NOTHING: this is what makes "`go run`
			// starts" true without a database, without a cache, without a
			// container (ADR 012).
			Default: driverLog,
			Drivers: map[string]config.Resources{
				// Written to the log: neither queryable, nor tamper-proof.
				driverLog: {},
				// Queryable, and kept beyond the retention of the logs.
				driverPostgres: {SQL: true},
			},
		},
	}
}
