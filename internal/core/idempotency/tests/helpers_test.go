// Package tests contains the BLACK BOX tests of the idempotency module: they
// use nothing but the public API, exactly like a caller.
//
// Repository convention (rules/tests.md): `{package}/tests/` for the black box,
// `{package}/internal_test.go` for the unexported identifiers. One file per
// test — the file name says what is verified, without having to open it.
package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// clock is a driven clock: an expiry test must never wait.
type clock struct {
	mu sync.Mutex
	at time.Time
}

func newClock() *clock {
	return &clock{at: time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)}
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

// newMemoryModule builds the module on its default driver, without any external
// dependency. An empty `ttl` lets the module's default value apply.
func newMemoryModule(t *testing.T, clk *clock, ttl string) idempotency.Module {
	t.Helper()
	cfg := config.Module{Enabled: true, Driver: "memory"}
	if ttl != "" {
		cfg.Options = map[string]any{"ttl": ttl}
	}
	mod, err := idempotency.New(cfg, idempotency.Deps{Now: clk.now})
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod
}

// request forges a complete request.
func request(key string, payload any) domain.Request {
	return domain.Request{Key: domain.Key(key), Fingerprint: domain.Fingerprint(payload)}
}
