// Package database carries the Postgres pool, the unit of work and the RLS
// scope.
//
// Central point of the package: the Querier function. A secondary adapter never
// receives the pool directly — it asks for the "querier" of the context, which
// is the transaction in progress if there is one, the pool otherwise. The same
// SQL code therefore works identically inside and outside a transaction, and it
// becomes impossible to write a query that escapes the open transaction.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Querier is the subset common to the pool and to a transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type contextKey struct{ name string }

// Context keys of the package.
//
// Globals owned up to: this is the Go idiom that makes any collision
// IMPOSSIBLE. The `contextKey` type is private to the package, so no other
// package can fabricate a key equal to these, even by copying the literal.
// Making them local or exported would break precisely the sought property, and
// a context key collision manifests itself as a transaction attributed to the
// wrong request — hence as a write into another customer's data.
//
//nolint:gochecknoglobals // context keys: the package-level private type IS the remedy against collisions
var (
	txKey     = &contextKey{name: "pgx-tx"}
	tenantKey = &contextKey{name: "tenant-id"}
)

// New opens the connection pool and checks that it answers.
//
// The check at start-up is deliberate: a service that starts without a database
// will signal its defect on the first user request, that is to say too late.
func New(ctx context.Context, cfg config.DB) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime.Duration()
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout.Duration()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("opening the pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout.Duration())
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database unreachable: %w", err)
	}
	return pool, nil
}

// WithTenant places the tenant scope into the context. RunInTx will translate
// it into a `SET LOCAL` so that the RLS policies apply.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// TenantFrom reads the tenant scope. Empty string if no scope is set.
func TenantFrom(ctx context.Context) string {
	tenantID, _ := ctx.Value(tenantKey).(string)
	return tenantID
}

// QuerierFrom returns the transaction of the context if there is one, the pool
// otherwise. It is the ONLY data access point of a secondary adapter: the same
// SQL therefore works identically inside and outside a transaction.
func QuerierFrom(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// InTx reports whether the context carries a transaction.
func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(txKey).(pgx.Tx)
	return ok
}

// RunInTx builds a unit of work shaped like a port.
//
// The rollback is triggered by a Result in Err — the business error is
// therefore transactionally significant, which is the expected behaviour: an
// email already taken must not leave an event in the outbox.
//
// An already open transaction is not nested: the function is executed inside
// the current transaction. That is what allows several transactional decorators
// to be composed without surprise.
func RunInTx[T, E any](
	pool *pgxpool.Pool,
) func(context.Context, func(context.Context) result.Result[T, E]) result.Result[T, E] {
	return func(ctx context.Context, fn func(context.Context) result.Result[T, E]) result.Result[T, E] {
		if InTx(ctx) {
			return fn(ctx)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			var zero E
			return result.Err[T, E](zero)
		}
		return runWithRollback(ctx, tx, fn)
	}
}

// runWithRollback isolates the commit/rollback mechanics.
func runWithRollback[T, E any](
	ctx context.Context,
	tx pgx.Tx,
	fn func(context.Context) result.Result[T, E],
) result.Result[T, E] {
	committed := false
	defer func() {
		if !committed {
			// The parent context may be cancelled: we roll back on a fresh
			// context, otherwise the rollback itself fails and the connection
			// stays dirty.
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()

	txCtx := context.WithValue(ctx, txKey, tx)
	if tenantID := TenantFrom(ctx); tenantID != "" {
		// SET LOCAL is bounded to the transaction: no state leaks towards the
		// pool, hence no risk that a following query inherits the tenant.
		if _, err := tx.Exec(txCtx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
			var zero E
			return result.Err[T, E](zero)
		}
	}

	res := fn(txCtx)
	if res.IsErr() {
		return res
	}
	if err := tx.Commit(txCtx); err != nil {
		var zero E
		return result.Err[T, E](zero)
	}
	committed = true
	return res
}

// TryAdvisoryLock attempts to take a session advisory lock.
//
// It is the election mechanism used by the scheduler: behind N replicas, only
// one obtains the lock and runs the task. The lock is released when the
// connection closes, so the death of a replica does not block the others
// durably.
func TryAdvisoryLock(ctx context.Context, q Querier, key int64) (bool, error) {
	var acquired bool
	if err := q.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&acquired); err != nil {
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}
	return acquired, nil
}

// ReleaseAdvisoryLock releases an advisory lock.
func ReleaseAdvisoryLock(ctx context.Context, q Querier, key int64) error {
	if _, err := q.Exec(ctx, "SELECT pg_advisory_unlock($1)", key); err != nil {
		return fmt.Errorf("pg_advisory_unlock: %w", err)
	}
	return nil
}

// Postgres error codes used for the translation into domain errors.
// A secondary adapter must never let a driver error go back up
// (rules/donnees-et-migrations.md §2).
const (
	CodeUniqueViolation      = "23505"
	CodeForeignKeyViolation  = "23503"
	CodeCheckViolation       = "23514"
	CodeQueryCanceled        = "57014"
	CodeSerializationFailure = "40001"
)

// PgErrorCode extracts the SQLSTATE code of an error, or the empty string.
func PgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// ConstraintName extracts the name of the violated constraint, or the empty
// string.
//
// It is that name which allows a uniqueness violation to be translated into a
// precise business error: a table can carry several unique constraints.
func ConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// IsNotFound reports a missing row. It is a nominal case, not a defect.
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// IsUnavailable reports a transient unavailability of the storage: to be
// translated into CodeUnavailable, never into CodeInternal (the two are not
// alerted on the same way).
func IsUnavailable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	switch PgErrorCode(err) {
	case CodeQueryCanceled, CodeSerializationFailure:
		return true
	default:
		return false
	}
}
