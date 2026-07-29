package idempotency

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares the drivers of this module — ADR 014.
//
// # Why here, and not in internal/config
//
// This table used to live in `internal/config/modules.go`, two packages away
// from the factory that actually builds the drivers. The comment that came with
// it already admitted to fearing divergence: "a typo in either of the two would
// make a module impossible to enable, with a message that blames the user's
// configuration".
//
// It now lives in the SAME package as the `switch` in `New`, often on the same
// screen. Divergence does not become impossible — nothing checks it
// mechanically, and ADR 014 notes this as its weakness [human] — but it becomes
// improbable.
//
// Replayable writes without side effects.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default requires NOTHING: that is what makes "`go run` starts"
			// true without a database, without a cache, without a container
			// (ADR 012).
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// `ttl` is read BEFORE the `switch` in `New`, hence by all three
				// drivers. `namespace` is read by redis only, where it prefixes
				// the keys: writing it elsewhere would have no effect at all, and
				// that is precisely the kind of ineffective setting one never
				// discovers (#93).
				//
				// Per instance: NO exclusivity behind several replicas.
				driverMemory: {Options: []string{OptionTTL}},
				// Exclusivity across replicas.
				driverPostgres: {SQL: true, Options: []string{OptionTTL}},
				// Exclusivity across replicas, passive expiry.
				driverRedis: {Cache: true, Options: []string{OptionTTL, OptionNamespace}},
			},
		},
	}
}
