package dynconf

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
// Flags and settings changeable at run time.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default requires NOTHING: this is what makes "`go run`
			// starts" true without a database, without a cache, without a
			// container (ADR 012).
			Default: driverFile,
			Drivers: map[string]config.Resources{
				// ⚠️ The two drivers do NOT admit the same options, and that is
				// exactly what per-driver declaration makes it possible to say.
				//
				// `flags` and `settings` carry the VERSIONED values: they only
				// make sense for the file driver. Writing them under the
				// postgres driver would be a silent design error — there, the
				// values live in the database, not in the repository.
				//
				// Read-only, reloaded from disk.
				driverFile: {Options: []string{OptionFlags, OptionSettings}},
				// Shared between replicas.
				driverPostgres: {SQL: true, Options: []string{OptionTTL}},
			},
		},
	}
}
