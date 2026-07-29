// Package modules assembles the catalogue of BUSINESS modules — ADR 014.
//
// Counterpart of the `internal/core` package, and for the same reason:
// `internal/config` may not name any module (`arch-go` rule 7), so the catalogue
// is built elsewhere and handed to it.
//
// # Why the catalogue belongs to the APPLICATION, not to the binary
//
// ADR 014 says "the composition root merges", which suggests that each binary
// declares what it mounts. Applied as such, that breaks — measured in CI, not
// deduced:
//
//	configuration: invalid configuration:
//	  modules.user_registration: unknown module
//
// `cmd/server` mounts `user_registration`; `cmd/worker` does not. But both read
// the SAME `config/modules.yaml`. A configuration valid for one binary therefore
// became invalid for the other — exactly the divergence this repository pays for
// every time one and the same truth lives in two places.
//
// The set of DECLARABLE modules is a property of the application, not of the
// binary that starts it. This file carries it, and both composition roots read
// it. Mounting a module remains a per-binary decision.
//
// # What this does not authorise
//
// Appearing in the catalogue mounts nothing. A module that is declarable but not
// mounted by a given binary is simply unused by it. Deny by default applies to
// the NAMES: what is not here is not configurable.
package modules

import (
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// Catalog returns the merged catalogue of this application's business modules.
//
// This is THE line an application adds to declare its `billing` module — and the
// only one. No framework file is touched: that is very exactly the friction
// ADR 014 removes.
func Catalog() (config.ModuleCatalog, error) {
	merged, err := config.MergeCatalogs(
		userregistration.Catalog(),
	)
	if err != nil {
		return nil, fmt.Errorf("business module catalogue: %w", err)
	}
	return merged, nil
}
