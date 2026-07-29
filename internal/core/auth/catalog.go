package auth

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares this module's drivers — ADR 014.
//
// No file of the framework names `auth`: the catalogue is built here, next to
// the `New` factory, and it SHARES its constants with that factory's `switch`.
// Divergence between what is declarable and what is constructible therefore
// becomes impossible.
//
// # Why a single driver for now
//
// ADR 017 declares the target — `postgres`, then `oidc` — and delivers only
// one. This repository has already paid for the opposite mistake: eight driver
// packages lived for months with ZERO tests, at any level (#37), and two
// production defects were asleep in them. A driver written with nothing to
// exercise it is code that looks like it works.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default demands NOTHING: that is what makes the promise
			// "`hexa new` then `go run`, and it starts" true — including with
			// authentication enabled.
			Default: driverMemory,
			Drivers: map[string]config.Resources{
				// Lost on restart, and local to the process: see the
				// NON-guarantees of the drivers/memory package before
				// considering it anywhere but in development.
				driverMemory: {Options: []string{OptionSessionTTL}},
			},
		},
	}
}
