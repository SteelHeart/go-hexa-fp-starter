// Package dynconf porte la configuration qui change SANS redéploiement :
// drapeaux de fonctionnalité et réglages métier.
//
// La ligne de partage avec le paquet config est nette : ce qui exige un
// redéploiement pour changer (DSN, ports, délais) est dans config ; ce qui doit
// pouvoir changer à chaud est ici. Un seuil métier en variable d'environnement
// ne se change jamais, donc il finit faux.
//
// Les drapeaux ne sont pas un luxe : le tronc unique (ADR 007) impose que le
// travail incomplet passe derrière un drapeau plutôt que derrière une branche
// longue. Sans cette brique, la règle est intenable.
package dynconf

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// FlagKey identifie un drapeau de fonctionnalité.
type FlagKey string

// SettingKey identifie un réglage métier.
type SettingKey string

// IsEnabled répond à « ce drapeau est-il actif ? ».
//
// Deny par défaut : un drapeau inconnu, une base injoignable ou une valeur
// illisible donnent false. Un drapeau qui s'activerait tout seul en cas de panne
// serait exactement l'inverse de ce qu'on veut.
type IsEnabled = func(ctx context.Context, key FlagKey) bool

// GetSetting lit un réglage métier. L'absence n'est pas une erreur.
type GetSetting = func(ctx context.Context, key SettingKey) fp.Option[string]

// Store lit les valeurs dynamiques avec un cache mémoire court.
//
// Le cache est indispensable : sans lui, un drapeau évalué dans une boucle
// produit une requête par itération.
type Store struct {
	querier   database.Querier
	ttl       time.Duration
	overrides map[string]string

	mu     sync.RWMutex
	cached map[string]entry
}

type entry struct {
	value   string
	present bool
	expires time.Time
}

// NewStore construit le magasin de configuration dynamique.
//
// `envPrefix` permet de surcharger n'importe quelle clé par variable
// d'environnement — indispensable en test, où l'on veut fixer un drapeau sans
// écrire en base. La surcharge gagne toujours sur la base.
func NewStore(q database.Querier, ttl time.Duration, envPrefix string) *Store {
	return &Store{
		querier:   q,
		ttl:       ttl,
		overrides: readOverrides(envPrefix),
		cached:    make(map[string]entry),
	}
}

// readOverrides collecte les surcharges d'environnement au démarrage.
// La config est immuable : on ne relit jamais l'environnement ensuite.
func readOverrides(prefix string) map[string]string {
	if prefix == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, raw := range os.Environ() {
		name, value, found := strings.Cut(raw, "=")
		if !found || !strings.HasPrefix(name, prefix) {
			continue
		}
		key := strings.ToLower(strings.TrimPrefix(name, prefix))
		out[strings.ReplaceAll(key, "_", ".")] = value
	}
	return out
}

// Flag retourne la fonction d'évaluation des drapeaux.
func (s *Store) Flag() IsEnabled {
	return func(ctx context.Context, key FlagKey) bool {
		raw, found := s.lookup(ctx, "flag", string(key))
		if !found {
			return false
		}
		enabled, err := strconv.ParseBool(raw)
		return err == nil && enabled
	}
}

// Setting retourne la fonction de lecture des réglages.
func (s *Store) Setting() GetSetting {
	return func(ctx context.Context, key SettingKey) fp.Option[string] {
		raw, found := s.lookup(ctx, "setting", string(key))
		if !found {
			return fp.None[string]()
		}
		return fp.Some(raw)
	}
}

// lookup applique l'ordre de priorité : surcharge d'environnement, puis cache,
// puis base.
func (s *Store) lookup(ctx context.Context, kind, key string) (string, bool) {
	full := kind + "." + key
	if value, found := s.overrides[full]; found {
		return value, true
	}

	s.mu.RLock()
	cached, found := s.cached[full]
	s.mu.RUnlock()
	if found && time.Now().Before(cached.expires) {
		return cached.value, cached.present
	}

	value, present := s.load(ctx, kind, key)
	s.mu.Lock()
	s.cached[full] = entry{value: value, present: present, expires: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return value, present
}

// load lit la valeur en base. Une panne donne « absent », donc désactivé.
func (s *Store) load(ctx context.Context, kind, key string) (string, bool) {
	const query = `SELECT value FROM dynamic_config WHERE kind = $1 AND key = $2`
	var value string
	if err := s.querier.QueryRow(ctx, query, kind, key).Scan(&value); err != nil {
		return "", false
	}
	return value, true
}

// Invalidate purge le cache. À appeler après une modification.
func (s *Store) Invalidate() {
	s.mu.Lock()
	s.cached = make(map[string]entry)
	s.mu.Unlock()
}

// Set écrit une valeur dynamique et purge le cache local.
//
// ⚠️ Le cache des AUTRES répliques n'est pas purgé : la propagation prend au
// plus le TTL. C'est un compromis assumé — l'alternative serait un canal
// d'invalidation, à ajouter le jour où le TTL devient trop long.
func (s *Store) Set(ctx context.Context, kind, key, value string) error {
	const query = `
		INSERT INTO dynamic_config (kind, key, value, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (kind, key) DO UPDATE SET value = excluded.value, updated_at = now()`
	if _, err := s.querier.Exec(ctx, query, kind, key, value); err != nil {
		return fmt.Errorf("écriture de la configuration dynamique %s.%s: %w", kind, key, err)
	}
	s.Invalidate()
	return nil
}

// StaticFlags construit une évaluation de drapeaux en mémoire, pour les tests.
// Elle conserve le « deny par défaut » : une clé absente donne false.
func StaticFlags(enabled map[FlagKey]bool) IsEnabled {
	return func(_ context.Context, key FlagKey) bool { return enabled[key] }
}

// StaticSettings construit une lecture de réglages en mémoire, pour les tests.
func StaticSettings(values map[SettingKey]string) GetSetting {
	return func(_ context.Context, key SettingKey) fp.Option[string] {
		value, found := values[key]
		if !found {
			return fp.None[string]()
		}
		return fp.Some(value)
	}
}
