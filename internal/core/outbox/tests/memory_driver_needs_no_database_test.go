package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestMemoryDriverNeedsNoDatabase verrouille la promesse centrale du socle :
// le pilote par défaut démarre sans aucune connexion.
func TestMemoryDriverNeedsNoDatabase(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: true, Driver: "memory"}, outbox.Deps{})
	if err != nil {
		t.Fatalf("le pilote memory ne doit exiger aucune dépendance: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); err != nil {
		t.Errorf("Enqueue sans base doit réussir: %v", err)
	}
}
