package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestDisabledModuleFailsLoudly : un module désactivé qui « marcherait quand
// même » est un piège. Un événement silencieusement ignoré ne se signale jamais.
func TestDisabledModuleFailsLoudly(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: false}, outbox.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("Enqueue sur module désactivé: attendu ErrDisabled, obtenu %v", err)
	}
	if err := mod.MarkDone(context.Background(), "x"); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("MarkDone sur module désactivé: attendu ErrDisabled, obtenu %v", err)
	}
}
