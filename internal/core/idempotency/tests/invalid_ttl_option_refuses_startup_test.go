package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
)

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
