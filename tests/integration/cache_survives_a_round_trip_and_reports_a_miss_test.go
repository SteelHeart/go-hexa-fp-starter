//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/cache"
)

// TestCacheSurvivesARoundTripAndReportsAMiss exercises the `cache` package,
// which had NO test at all — not even compilation exercised (#37).
//
// The reason for that absence is structural: `cache.New` performs a `Ping` at
// construction and `cache.JSON` requires a client. This package is therefore
// only reachable at this level, never by `go test ./...`.
//
// What the test guards: the round trip of a typed value, and above all the
// behaviour on ABSENCE. `Getter` treats absence and failure the same way — an
// empty option — because in both cases the caller does not have the value and
// has to fall back on the source of truth. That choice is deliberate, and it
// can only be verified against a real Redis.
func TestCacheSurvivesARoundTripAndReportsAMiss(t *testing.T) {
	ctx := ctxTest(t)
	client := redisClient(t)

	type profile struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	namespace := unique(t, "integration-cache")
	get, set, del := cache.JSON[profile](client, namespace, time.Minute)

	// Absence BEFORE any write: the option must be empty.
	if absent := get(ctx, "unknown"); absent.IsSome() {
		t.Fatal("a key never written must return an empty option")
	}

	// `Setter` and `Deleter` return NOTHING, and that is a decision: a cache
	// that falls over must never make a business request fail. The consequence
	// is that a silent write is indistinguishable from a successful one — hence
	// the read-back that follows, the only proof available.
	want := profile{Name: "alice", Age: 30}
	set(ctx, "key", want)
	t.Cleanup(func() { del(ctxTest(t), "key") })

	read := get(ctx, "key")
	if read.IsNone() {
		t.Fatal("the written value must be read back: otherwise the cache caches nothing")
	}
	if got := read.ValueOr(profile{}); got != want {
		t.Errorf("value read back = %+v, want %+v", got, want)
	}

	// After deletion, back to absence.
	del(ctx, "key")
	if after := get(ctx, "key"); after.IsSome() {
		t.Fatal("a deleted key must become absent again — an invalidation that " +
			"invalidates nothing is worse than no cache at all")
	}
}
