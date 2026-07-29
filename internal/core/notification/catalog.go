package notification

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares the drivers of this module — ADR 014.
//
// No file of the framework names `notification`: the catalogue is built here,
// next to the `New` factory, and it SHARES its constants with it. Divergence
// between what is declarable and what is constructible therefore becomes
// impossible.
//
// # A single driver, and it sends nothing
//
// `smtp`, `mailjet`, `ses` are described in
// documentation/technique/modules-noyau.md and are NOT written. This repository
// has already paid the opposite mistake: eight driver packages lived for months
// with zero tests, at any level (#37), and two production defects were sleeping
// in there. An SMTP driver written without a server to exercise it is code that
// looks like it works — and on this module, "looks like it works" means nobody
// receives anything for weeks.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// The default requires NOTHING: no SMTP server, no account at a
			// provider. That is what lets the complete chain — registration,
			// outbox, relay, notification — start from a `go run`.
			Default: driverLog,
			Drivers: map[string]config.Resources{
				// Sends to nobody: see the NON-GUARANTEES of the drivers/log
				// package before considering it anywhere but in development.
				driverLog: {Options: []string{OptionBody}},
			},
		},
	}
}
