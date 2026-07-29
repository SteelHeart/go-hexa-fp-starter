package userregistration

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog declares this module's drivers — ADR 014.
//
// # THIS file is what answers the friction
//
// Before ADR 014, a business module could not appear in `config/modules.yaml`:
// the configuration refused its name with
// `modules.user_registration: unknown module`, because the only module table
// lived in `internal/config/modules.go` — a framework file.
//
// Measured consequence: `cmd/server` read `cfg.Modules[Name].Driver`, a field
// that ALWAYS stayed empty. The reference slice was therefore silently
// bypassing the very mechanism it is supposed to demonstrate — and its shape is
// the one that will be copied to write `billing`.
//
// This file is the demonstration: a business module declares its drivers at
// home, the composition root merges them, and NO framework file names
// `user_registration`. Writing `billing` calls for the same file, in the same
// place, without touching the starter.
//
// The module stays removable with a single `rm -rf`: this file leaves with it,
// and the only line left to remove is the composition root line that mounts it.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// `memory` is the default, and that is a decision: it requires no
			// infrastructure, so `go run` starts. See the NON-GUARANTEES of the
			// drivers/memory package before considering it anywhere other than
			// development.
			Default: DriverMemory,
			Drivers: map[string]config.Resources{
				// Lost on restart, and local to the process.
				DriverMemory: {},
			},
		},
	}
}
