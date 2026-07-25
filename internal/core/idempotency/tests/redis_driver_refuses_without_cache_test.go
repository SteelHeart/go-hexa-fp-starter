package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

// TestRedisDriverRefusesWithoutCache : même exigence pour le cache.
func TestRedisDriverRefusesWithoutCache(t *testing.T) {
	t.Parallel()

	_, err := idempotency.New(
		config.Module{Enabled: true, Driver: "redis"}, idempotency.Deps{})
	if !errors.Is(err, idempotency.ErrCacheRequired) {
		t.Errorf("erreur = %v, attendu ErrCacheRequired", err)
	}
}
