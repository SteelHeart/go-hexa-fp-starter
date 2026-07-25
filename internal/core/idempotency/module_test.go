package idempotency_test

import (
	"context"
	"errors"
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

// TestMemoryDriverNeedsNoDatabaseNorCache verrouille la promesse de l'ADR 012 :
// `hexa new` puis `go run` doit démarrer sans base, sans Redis, sans Docker.
func TestMemoryDriverNeedsNoDatabaseNorCache(t *testing.T) {
	t.Parallel()

	mod, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memory"},
		idempotency.Deps{}, // ni Pool, ni Cache, ni horloge
	)
	if err != nil {
		t.Fatalf("le pilote par défaut ne doit réclamer aucune dépendance: %v", err)
	}
	if _, err := mod.Reserve(context.Background(), request("k1", "charge")); err != nil {
		t.Errorf("réservation sur le pilote mémoire: %v", err)
	}
}

// TestDisabledModuleRefusesLoudly : un module désactivé échoue explicitement.
// Une idempotence inerte laisserait passer les doublons sans se signaler.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := idempotency.New(config.Module{Enabled: false, Driver: "memory"}, idempotency.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", "charge")); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Reserve = %v, attendu ErrDisabled", err)
	}
	if err := mod.Complete(ctx, "k1", nil); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Complete = %v, attendu ErrDisabled", err)
	}
	if err := mod.Release(ctx, "k1"); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Release = %v, attendu ErrDisabled", err)
	}
	if _, err := mod.Purge(ctx); !errors.Is(err, idempotency.ErrDisabled) {
		t.Errorf("Purge = %v, attendu ErrDisabled", err)
	}
}

// TestPostgresDriverRefusesWithoutPool : un pilote qui exige une base sans base
// refuse au démarrage, jamais à la première requête.
func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "postgres"}, idempotency.Deps{})
	if !errors.Is(err, idempotency.ErrPoolRequired) {
		t.Errorf("erreur = %v, attendu ErrPoolRequired", err)
	}
}

// TestRedisDriverRefusesWithoutCache : même exigence pour le cache.
func TestRedisDriverRefusesWithoutCache(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "redis"}, idempotency.Deps{})
	if !errors.Is(err, idempotency.ErrCacheRequired) {
		t.Errorf("erreur = %v, attendu ErrCacheRequired", err)
	}
}

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique. La
// validation de configuration a déjà rejeté le pilote ; ce second refus garantit
// qu'aucun chemin ne contourne le premier.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memcached"}, idempotency.Deps{})
	if err == nil {
		t.Fatal("un pilote inconnu doit refuser le démarrage")
	}
}

// TestInvalidTTLOptionRefusesStartup : une option illisible refuse le démarrage.
// Se rabattre silencieusement sur la valeur par défaut donnerait un TTL surprise,
// donc une fenêtre de rejeu qui n'est pas celle qu'on croit.
func TestInvalidTTLOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	cases := map[string]any{
		"unité manquante": "24",
		"négative":        "-1h",
		"nulle":           "0s",
		"booléen":         true,
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Module{
				Enabled: true, Driver: "memory",
				Options: map[string]any{"ttl": value},
			}
			if _, err := idempotency.New(cfg, idempotency.Deps{}); err == nil {
				t.Errorf("ttl=%v doit refuser le démarrage", value)
			}
		})
	}
}
