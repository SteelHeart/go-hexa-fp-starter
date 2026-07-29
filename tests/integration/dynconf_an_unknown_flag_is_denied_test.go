//go:build integration

package integration

import (
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
	pgdyn "github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/postgres"
)

// TestDynconfAnUnknownFlagIsDenied exercises deny-by-default all the way down
// into the database.
//
// It is the most important property of this module and the easiest to lose: a
// missing flag, an unreachable database, a query in error — all of them must
// return OFF. Returning "on" on a failure would enable a feature nobody asked
// for, at the worst possible moment.
//
// The test checks both directions against the real table: missing = off,
// written = on after cache invalidation.
//
// The TTL is set to zero: this module caches values, and without that the test
// would measure the freshness of the cache rather than the content of the
// database.
func TestDynconfAnUnknownFlagIsDenied(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	logger := slog.New(slog.DiscardHandler)
	store := pgdyn.New(p, logger, 0, time.Now)

	absent := domain.FlagKey(unique(t, "integration-absent"))
	if store.Flag(ctx, absent) {
		t.Fatal("a MISSING flag must be off: deny by default")
	}

	present := domain.FlagKey(unique(t, "integration-present"))
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t),
			"DELETE FROM platform.dynamic_config WHERE key = $1", string(present))
	})

	if err := store.Set(ctx, domain.Change{
		Kind:  domain.KindFlag,
		Key:   string(present),
		Value: "true",
	}); err != nil {
		t.Fatalf("writing the flag: %v", err)
	}
	store.Invalidate()

	if !store.Flag(ctx, present) {
		t.Fatal("a flag written to `true` must be on: otherwise no hot toggle " +
			"works, and the module is good for nothing")
	}

	// Rewriting to `false`: the toggle has to work both ways. A module that
	// only knows how to switch on does not allow switching off during an
	// incident.
	if err := store.Set(ctx, domain.Change{
		Kind:  domain.KindFlag,
		Key:   string(present),
		Value: "false",
	}); err != nil {
		t.Fatalf("rewriting the flag: %v", err)
	}
	store.Invalidate()

	if store.Flag(ctx, present) {
		t.Fatal("a flag toggled back to `false` must be off — that is the " +
			"direction taken during an incident")
	}
}
