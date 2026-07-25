package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

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
