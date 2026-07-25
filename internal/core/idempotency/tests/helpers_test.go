// Package tests contient les tests en BOÎTE NOIRE du module idempotency : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// clock est une horloge pilotée : un test d'expiration ne doit jamais attendre.
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

// newMemoryModule construit le module sur son pilote par défaut, sans aucune
// dépendance externe. Un `ttl` vide laisse jouer la valeur par défaut du module.
func newMemoryModule(t *testing.T, clk *clock, ttl string) idempotency.Module {
	t.Helper()
	cfg := config.Module{Enabled: true, Driver: "memory"}
	if ttl != "" {
		cfg.Options = map[string]any{"ttl": ttl}
	}
	mod, err := idempotency.New(cfg, idempotency.Deps{Now: clk.now})
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod
}

// request forge une requête complète.
func request(key string, payload any) domain.Request {
	return domain.Request{Key: domain.Key(key), Fingerprint: domain.Fingerprint(payload)}
}
