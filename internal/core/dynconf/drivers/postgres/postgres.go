// Package postgres implements dynamic configuration on PostgreSQL.
//
// # GUARANTEES
//
//   - **Changeable at run time**: this is the reason the module exists, and the
//     only shipped driver that holds it.
//   - **Shared between replicas**: a write is visible to all of them.
//
// # NON-GUARANTEES
//
//   - **Propagation takes up to one TTL.** `Set` only purges the LOCAL cache;
//     the other replicas keep the previous value until it expires. A flag
//     switched off in an emergency therefore takes at most `ttl` to go off
//     everywhere. The alternative — an invalidation channel — will be justified
//     the day that delay becomes unacceptable.
//   - **A database outage switches the flags off**, by design (see
//     ports.IsEnabled). The driver logs it, since it cannot return it.
//   - **Absences are cached** just like presences: without that, a key never set
//     would cost one query per evaluation.
//
// # State
//
// Written, NEVER run against a database: the migration of
// `platform.dynamic_config` does not exist yet (issue #2).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// cached is a memorised value, present or absent.
type cached struct {
	value   string
	found   bool
	expires time.Time
}

// Store reads the dynamic values with a short in-memory cache.
//
// The cache is indispensable, not an optimisation: a flag evaluated in a loop
// would otherwise produce one query per iteration.
type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	ttl    time.Duration
	now    func() time.Time

	mu     sync.RWMutex
	values map[string]cached
}

// New builds the store.
func New(pool *pgxpool.Pool, logger *slog.Logger, ttl time.Duration, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{
		pool:   pool,
		logger: logger,
		ttl:    ttl,
		now:    now,
		values: make(map[string]cached),
	}
}

// Flag implements ports.IsEnabled.
func (s *Store) Flag(ctx context.Context, key domain.FlagKey) bool {
	entry := s.lookup(ctx, domain.KindFlag, string(key))
	if !entry.Found {
		return false
	}
	return domain.ParseFlag(entry.Value)
}

// Setting implements ports.GetSetting.
func (s *Store) Setting(ctx context.Context, key domain.SettingKey) domain.Setting {
	return s.lookup(ctx, domain.KindSetting, string(key))
}

// lookup applies the order: live cache, then database.
func (s *Store) lookup(ctx context.Context, kind domain.Kind, key string) domain.Setting {
	full := domain.Qualify(kind, key)
	now := s.now()

	s.mu.RLock()
	entry, known := s.values[full]
	s.mu.RUnlock()
	if known && now.Before(entry.expires) {
		return domain.Setting{Value: entry.value, Found: entry.found}
	}

	value, found := s.load(ctx, kind, key)
	s.mu.Lock()
	s.values[full] = cached{value: value, found: found, expires: now.Add(s.ttl)}
	s.mu.Unlock()
	return domain.Setting{Value: value, Found: found}
}

// load reads the value from the database.
//
// An outage gives "absent", hence a switched-off flag. It is logged here
// because the contract of the port forbids returning it: without that trace, an
// unreachable database would switch the hidden features off in silence.
func (s *Store) load(ctx context.Context, kind domain.Kind, key string) (string, bool) {
	const query = `SELECT value FROM platform.dynamic_config WHERE kind = $1 AND key = $2`

	var value string
	err := s.pool.QueryRow(ctx, query, string(kind), key).Scan(&value)
	switch {
	case err == nil:
		return value, true
	case errors.Is(err, pgx.ErrNoRows):
		return "", false
	default:
		s.logger.ErrorContext(ctx, "dynamic configuration unreadable, falling back on absence",
			slog.String("kind", string(kind)),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return "", false
	}
}

// Set implements ports.Set.
func (s *Store) Set(ctx context.Context, change domain.Change) error {
	if !change.IsValid() {
		return fmt.Errorf("%w: %s", domain.ErrInvalidChange, change.Describe())
	}

	const query = `
		INSERT INTO platform.dynamic_config (kind, key, value, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (kind, key) DO UPDATE
			SET value = excluded.value, updated_at = now()`

	_, err := s.pool.Exec(ctx, query, string(change.Kind), change.Key, change.Value)
	if err != nil {
		return fmt.Errorf("writing the dynamic configuration %s: %w", change.Describe(), err)
	}
	s.Invalidate()
	return nil
}

// Invalidate implements ports.Invalidate.
func (s *Store) Invalidate() {
	s.mu.Lock()
	s.values = make(map[string]cached)
	s.mu.Unlock()
}
