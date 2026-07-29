//go:build integration

// Package integration exercises the drivers against the REAL SERVICES.
//
// # What this level adds, and why it was not optional
//
// Eight driver packages had zero tests, at any level (#37). The `memory`
// driver of `idempotency` has twenty-four tests proving exclusivity; the
// `postgres` driver, the one that runs in PRODUCTION, had none.
//
// This was not an oversight but a consequence: those drivers require a
// service, so `go test ./...` without a tag cannot reach them. The
// `integration` tag was carried by NO file in the repository, and the CI had
// no matching job — the debt was named, never paid.
//
// # What this level does NOT do
//
// It does not replay the domain tests. The pure properties — refusal of an
// empty key, bounds of the exponential backoff, normalisation — are already
// verified in microseconds under `{module}/tests/`. Repeating them here would
// cost a service to learn nothing.
//
// It verifies what they CANNOT verify: that the SQL is valid, that the
// migration constraints do what we believe they do, and above all that the
// CONCURRENCY guarantees hold — a `FOR UPDATE SKIP LOCKED` or an advisory lock
// can only be tested against a real engine.
//
// # No `t.Skip`, ever
//
// A skipped test looks in every respect like a passing one: that is the very
// shape ADR 013 fights. If the service is unreachable, these tests FAIL while
// naming the command that starts it. The false green has no place here, and
// least of all at the level that guards the production code.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// dsnPostgres returns the migration DSN, the only role allowed to create.
//
// `DB_MIGRATION_DSN` and not `DB_DSN`: the application role has neither
// CREATE, nor ALTER, nor DROP (ADR 011 §4). These tests clean up their own
// rows, which requires rights the application role must NOT have.
func dsnPostgres(t *testing.T) string {
	t.Helper()
	if dsn := os.Getenv("DB_MIGRATION_DSN"); dsn != "" {
		return dsn
	}
	// The same default as `config/env/development.yaml`. Do NOT fail here: a
	// missing variable is not a missing service, and it is the Ping that
	// decides. Failing on the variable would send one looking in the wrong
	// place.
	return "postgres://hexa:hexa@localhost:5432/hexa?sslmode=disable"
}

// pool opens a Postgres pool and CHECKS that it answers.
//
// The `Ping` is indispensable: `pgxpool.New` is lazy and succeeds against a
// dead address. Without it, the first failure would occur in the middle of a
// test, and would accuse the query rather than the missing service.
func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := pgxpool.New(ctx, dsnPostgres(t))
	if err != nil {
		t.Fatalf("opening the Postgres pool: %v", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		t.Fatalf("Postgres unreachable (%v) — start the stack with `task up`", err)
	}
	t.Cleanup(p.Close)
	return p
}

// redisClient opens a Redis client and checks that it answers.
func redisClient(t *testing.T) *redis.Client {
	t.Helper()

	// Same default as `config/data.yaml`.
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("Redis unreachable (%v) — start the stack with `task up`", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// unique returns an identifier specific to a test.
//
// The tests at this level share ONE database: they run in parallel and they
// get replayed. Two tests writing the same key would contaminate each other,
// and the symptom — an intermittent failure — costs more to diagnose than the
// defect it hides.
func unique(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%s-%d", prefix, t.Name(), time.Now().UnixNano())
}

// ctxTest returns a bounded context: a test waiting on a lock must not block
// the whole suite.
func ctxTest(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}
