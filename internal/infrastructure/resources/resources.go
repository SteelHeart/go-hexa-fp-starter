// Package resources opens the connections that the ENABLED modules require,
// and no others.
//
// # The defect this package fixes — #103
//
// `database.New` was called by NO binary. Neither `cmd/server`, nor
// `cmd/worker`. Measured consequence: the dispatcher refused the `memory`
// driver by design — it would dispatch a store the server does not share — and
// could not use `postgres` for lack of a pool. **It started in no
// configuration at all.**
//
// By extension, no `postgres` driver of the repository was reachable from a
// binary: `audit`, `dynconf`, `idempotency`, `outbox`, `scheduler` all have
// one, all unreachable. They have integration tests (#37) and nothing mounted
// them.
//
// Nobody had seen it because the only path ever exercised was the REFUSAL one:
// the CI job checks that the dispatcher refuses an unshared outbox, which is a
// good guard — but the nominal path was executed nowhere.
//
// # Why this package, and not five lines in each composition root
//
// Because there are TWO of them, and this repository has already paid three
// times for the divergence between `cmd/server` and `cmd/worker`. A pool
// opening that differs from one binary to the other produces a server that
// writes and a dispatcher that does not read — exactly the defect that has just
// been fixed, under another form.
//
// # What this package does NOT do
//
// It opens nothing "just in case". That is the promise of ADR 012: with the
// shipped configuration, every driver is dependency-free, so **no connection is
// opened** and `go run ./cmd/server` starts on a bare machine. Fixing #103 was
// not to cost that promise.
package resources

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/cache"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
)

// Connections carries what has been opened, and the means to close it again.
//
// A struct rather than three returns: `Open` would return
// `(*pgxpool.Pool, *goredis.Client, error)`, and the architecture rule refuses
// it. It is right here for a reason proper to the subject — this is the FIFTH
// occurrence of the same lesson in this repository, after `election`,
// `decodedHash`, `RetryPolicy`, `messaging.Broker` and `worker`.
//
// ⚠️ **Both fields can be nil, legitimately.** A nil does not signal a failure:
// it signals that no enabled module required that resource. The modules that do
// not need it receive that nil without ever dereferencing it, and those that do
// need it refuse to start, saying so.
type Connections struct {
	Pool  *pgxpool.Pool
	Cache *goredis.Client
}

// Open opens what the configuration requires, and nothing more.
//
// The decision comes from `config.Modules.RequiresSQL` and `RequiresCache`,
// which only interrogate the ENABLED modules and the driver actually retained.
// Those two functions existed and were called by no binary: the "starts without
// a database" promise was *asserted* by tests, never *exercised*.
//
// Opening CHECKS that the connection answers — `database.New` does a ping, and
// so does `cache.New`. That is deliberate: a service that starts without a
// database signals its defect on the first user request, that is to say too
// late, and from a place that does not blame the right cause.
func Open(ctx context.Context, cfg config.Config, catalog config.ModuleCatalog) (Connections, error) {
	var opened Connections

	if cfg.Modules.RequiresSQL(catalog) {
		pool, err := database.New(ctx, cfg.Database)
		if err != nil {
			return Connections{}, fmt.Errorf("an enabled module requires a database: %w", err)
		}
		opened.Pool = pool
	}

	if cfg.Modules.RequiresCache(catalog) {
		client, err := cache.New(ctx, cfg.Cache)
		if err != nil {
			// The already opened pool is closed before returning: without that,
			// a start-up that fails halfway would leave connections dangling,
			// and a restart loop would accumulate them until the database
			// server saturates.
			opened.Close()
			return Connections{}, fmt.Errorf("an enabled module requires a cache: %w", err)
		}
		opened.Cache = client
	}

	return opened, nil
}

// Close closes again what has been opened. Safe on a zero value.
//
// It does NOT return an error, deliberately. It is called in a `defer` at
// shutdown time, where there is no longer anyone to account to: a caller
// receiving an error could only ignore or log it, and the obligation to handle
// it pushes towards writing `defer func() { _ = c.Close() }()` — hence towards
// hiding, rather than deciding.
func (c Connections) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
	if c.Cache != nil {
		_ = c.Cache.Close()
	}
}
