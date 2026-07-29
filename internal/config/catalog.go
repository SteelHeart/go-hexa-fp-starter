package config

// Module catalogue — ADR 014.
//
// # This file defines TYPES, never content
//
// No module name is written here, and that is the whole decision. This package
// carried two tables — `knownDrivers` and `defaultDrivers` — which enumerated
// the six core modules. An absent module was refused.
//
// Excellent deny-by-default. For the starter. For an application, it meant:
// declaring your own `billing` module forces you to modify a file of the
// framework. Measured, not deduced — the binary answered
// `modules.billing: unknown module` with a return code of 1.
//
// Rule 7 of `arch-go` already forbids this package from depending on an
// internal package. It was respected to the letter and circumvented in spirit:
// the package depended on no module, but it NAMED them. The catalogue is now
// RECEIVED, built by the composition root, the only code allowed to know
// everything (ADR 004).
//
// An empty catalogue refuses EVERYTHING. Deny-by-default does not loosen: it
// changes source.

import (
	"fmt"
	"slices"
	"sort"
)

// Resources says what a driver requires of the process that hosts it.
//
// This is what allows a binary to open a connection only if it needs one, and
// therefore to start without a database when every active driver lives in
// memory or on file — the promise of ADR 012.
//
// The zero value requires NOTHING, and that is the right default: a driver that
// forgets to declare its needs opens no connection, it fails at its
// construction. The opposite would open a useless connection, silently.
type Resources struct {
	// SQL: a relational database, whatever the ENGINE. No engine is imposed by
	// the starter (ADR 012): `postgres` is one driver among others.
	SQL bool
	// Cache: a network cache shared between replicas.
	Cache bool
	// Options enumerates the keys this driver READS in `modules.<name>.options`.
	//
	// Any other key refuses to start. Without this list, "absent" and
	// "misspelt" are indistinguishable: the option accessors return the default
	// value in both cases, and the driver starts with a setting nobody asked
	// for.
	//
	// Measured before being fixed (#93): `bath_size` instead of `batch_size`
	// let the server start, mount the module and say nothing about it.
	//
	// The zero value admits NO option, and that is the right default: a driver
	// that forgets to declare its own sees its configuration refused, which
	// gets noticed. The opposite would reopen the hole for every driver at
	// once.
	Options []string
}

// DriverSet declares the drivers of ONE module.
//
// `Drivers` carries both the list of admitted drivers and what each one
// requires: a single table, hence no risk of declaring a driver without saying
// what it consumes, nor the other way round.
type DriverSet struct {
	// Default is the driver retained when the configuration names none.
	//
	// It MUST be the one that requires nothing: this is what makes the promise
	// "`hexa new` then `go run`, and it starts" true. Never the most complete
	// one.
	Default string
	// Drivers enumerates the admitted drivers and their needs.
	Drivers map[string]Resources
}

// ModuleCatalog associates a module name with its drivers.
//
// It is built by the composition root, which merges the catalogue of every
// mounted module — core as well as business. A module that is not mounted does
// not appear in it, and is therefore not configurable: one does not configure
// what one has not plugged in.
type ModuleCatalog map[string]DriverSet

// AllowedDrivers returns the admitted drivers of a module, sorted.
//
// Sorted because this list serves to compose an error message: a map order is
// random, and a message that changes on every run makes it impossible to
// compare two traces.
func (c ModuleCatalog) AllowedDrivers(module string) []string {
	set, known := c[module]
	if !known {
		return nil
	}
	names := make([]string, 0, len(set.Drivers))
	for name := range set.Drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultDriver returns the default driver of a module, or the empty string.
func (c ModuleCatalog) DefaultDriver(module string) string { return c[module].Default }

// AllowedOptions returns the option keys admitted by a driver, sorted.
//
// Sorted for the same reason as AllowedDrivers: this list composes an error
// message, and a map order changes on every run — which makes it impossible to
// compare two traces.
func (c ModuleCatalog) AllowedOptions(module, driver string) []string {
	allowed := slices.Clone(c.Requires(module, driver).Options)
	slices.Sort(allowed)
	return allowed
}

// Requires returns the needs of a driver of a module.
//
// An unknown driver requires nothing: it is validation that refuses it, not
// this accessor. Returning `SQL: true` for a non-existent driver would open a
// connection in the name of a driver we are precisely about to refuse.
func (c ModuleCatalog) Requires(module, driver string) Resources {
	return c[module].Drivers[driver]
}

// MergeCatalogs assembles the catalogues of the mounted modules.
//
// Two modules cannot claim the same name: that would be a silent collision,
// where one of the two would end up configured by the drivers of the other.
// Explicit refusal — deny by default.
//
// It is the composition root that calls this, and it alone: it is the only code
// that knows the list of mounted modules (ADR 004).
func MergeCatalogs(catalogs ...ModuleCatalog) (ModuleCatalog, error) {
	merged := ModuleCatalog{}
	for _, catalog := range catalogs {
		for name, set := range catalog {
			if _, collision := merged[name]; collision {
				return nil, fmt.Errorf(
					"catalogue: module %q is declared twice — two modules cannot carry the same name", name)
			}
			merged[name] = set
		}
	}
	return merged, nil
}
