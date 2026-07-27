// Package postgres implémente la configuration dynamique sur PostgreSQL.
//
// # GARANTIES
//
//   - **Modifiable à chaud** : c'est la raison d'être du module, et le seul pilote
//     livré qui la tienne.
//   - **Partagé entre répliques** : une écriture est visible par toutes.
//
// # NON-GARANTIES
//
//   - **La propagation prend jusqu'à un TTL.** `Set` ne purge que le cache LOCAL ;
//     les autres répliques gardent la valeur précédente jusqu'à expiration. Un
//     drapeau éteint en urgence met donc au plus `ttl` à s'éteindre partout.
//     L'alternative — un canal d'invalidation — se justifiera le jour où ce délai
//     deviendra inacceptable.
//   - **Une panne de base éteint les drapeaux**, par conception (voir
//     ports.IsEnabled). Le pilote la journalise, puisqu'il ne peut pas la
//     retourner.
//   - **Les absences sont mises en cache** au même titre que les présences : sans
//     ça, une clé jamais posée coûterait une requête par évaluation.
//
// # État
//
// Écrit, JAMAIS exécuté contre une base : la migration de
// `platform.dynamic_config` n'existe pas encore (issue #2).
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

// cached est une valeur mémorisée, présente ou absente.
type cached struct {
	value   string
	found   bool
	expires time.Time
}

// Store lit les valeurs dynamiques avec un cache mémoire court.
//
// Le cache est indispensable, pas une optimisation : un drapeau évalué dans une
// boucle produirait sinon une requête par itération.
type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	ttl    time.Duration
	now    func() time.Time

	mu     sync.RWMutex
	values map[string]cached
}

// New construit le magasin.
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

// Flag implémente ports.IsEnabled.
func (s *Store) Flag(ctx context.Context, key domain.FlagKey) bool {
	entry := s.lookup(ctx, domain.KindFlag, string(key))
	if !entry.Found {
		return false
	}
	return domain.ParseFlag(entry.Value)
}

// Setting implémente ports.GetSetting.
func (s *Store) Setting(ctx context.Context, key domain.SettingKey) domain.Setting {
	return s.lookup(ctx, domain.KindSetting, string(key))
}

// lookup applique l'ordre : cache vivant, puis base.
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

// load lit la valeur en base.
//
// Une panne donne « absent », donc drapeau éteint. Elle est journalisée ici parce
// que le contrat du port interdit de la retourner : sans cette trace, une base
// injoignable éteindrait les fonctionnalités masquées en silence.
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
		s.logger.ErrorContext(ctx, "configuration dynamique illisible, repli sur l'absence",
			slog.String("kind", string(kind)),
			slog.String("key", key),
			slog.String("error", err.Error()),
		)
		return "", false
	}
}

// Set implémente ports.Set.
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
		return fmt.Errorf("écriture de la configuration dynamique %s: %w", change.Describe(), err)
	}
	s.Invalidate()
	return nil
}

// Invalidate implémente ports.Invalidate.
func (s *Store) Invalidate() {
	s.mu.Lock()
	s.values = make(map[string]cached)
	s.mu.Unlock()
}
