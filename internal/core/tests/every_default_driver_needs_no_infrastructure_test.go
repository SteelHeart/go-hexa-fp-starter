package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
)

// TestEveryDefaultDriverNeedsNoInfrastructure holds the central promise of
// ADR 012: `hexa new` then `go run`, and it starts.
//
// Every core module enabled at once, each on the default driver of ITS OWN
// catalogue: the result must require neither a database nor a cache.
//
// The day someone makes `postgres` the default of a module — out of
// convenience, because it is the most complete driver — this test fails. It is
// the only place that will notice before the users do.
//
// It used to live in `internal/config/internal_test.go`, where it queried a
// framework table. Since ADR 014 there is no table any more: it therefore
// queries the REAL catalogues of every core module, which is what it should have
// done from the start.
func TestEveryDefaultDriverNeedsNoInfrastructure(t *testing.T) {
	t.Parallel()

	catalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("core catalogue: %v", err)
	}

	all := config.Modules{}
	for name := range catalog {
		all[name] = config.Module{Enabled: true}
	}
	all = all.Resolve(catalog)

	if all.RequiresSQL(catalog) {
		t.Error("default drivers must require no database")
	}
	if all.RequiresCache(catalog) {
		t.Error("default drivers must require no cache")
	}
}
