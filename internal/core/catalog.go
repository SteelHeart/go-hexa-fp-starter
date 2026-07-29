// Package core assembles the catalogue of the CORE modules — ADR 014.
//
// It holds nothing else, deliberately: it is the only place in the starter that
// names its eight modules, and all it does is enumerate them.
//
// # Why this package exists
//
// `internal/config` cannot name any module — `arch-go` rule 7, which forbids it
// any internal dependency. The catalogue must therefore be built elsewhere and
// handed to it.
//
// The composition root could do it, but `cmd/server` and `cmd/worker` would then
// write the same list twice — and the day they diverged, a module would be
// configurable in one binary and not in the other, without anything saying so.
// That is the divergence this repository has already paid for three times.
//
// # What this catalogue declares, and what it does not
//
// It declares what the starter's binaries EMBED, not what they mount. The
// nuance matters: `cmd/server` currently wires only `outbox` and `auth`, while
// `config/modules.yaml` declares them all. Restricting the catalogue to the
// wired modules alone would therefore make it REFUSE the shipped configuration.
//
// ADR 014 states the rule more narrowly — "a module that is not mounted is not
// in the catalogue". The gap is assumed, and written here rather than left
// unsaid: the property that matters is that no ARBITRARY name gets through, and
// that one holds.
package core

import (
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// Catalog returns the merged catalogue of the core modules.
//
// Every module declares its own at home, next to the factory that builds its
// drivers. This file only brings them together: it holds no driver name, so it
// cannot diverge from a factory.
//
// A name collision is impossible here — eight distinct constants — but
// `MergeCatalogs` would refuse it, and that is the guarantee that matters the
// day an application adds its own.
func Catalog() (config.ModuleCatalog, error) {
	merged, err := config.MergeCatalogs(
		outbox.Catalog(),
		idempotency.Catalog(),
		dynconf.Catalog(),
		audit.Catalog(),
		storage.Catalog(),
		scheduler.Catalog(),
		auth.Catalog(),
		notification.Catalog(),
	)
	if err != nil {
		return nil, fmt.Errorf("core catalogue: %w", err)
	}
	return merged, nil
}
