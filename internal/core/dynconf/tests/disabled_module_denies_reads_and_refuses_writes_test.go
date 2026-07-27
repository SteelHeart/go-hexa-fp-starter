package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// TestDisabledModuleDeniesReadsAndRefusesWrites documente la seule exception du
// dépôt à « un module désactivé échoue bruyamment ».
//
// La lecture ne peut PAS échouer : ports.IsEnabled ne retourne pas d'erreur. La
// seule réponse possible, `false`, est justement celle de « deny par défaut » —
// l'inertie coïncide ici avec le refus, ce qui n'est le cas d'aucun autre module.
//
// L'écriture, elle, PEUT parler : elle refuse. Un appelant qui croirait avoir
// changé un drapeau sur un module éteint est le seul vrai piège possible ici.
func TestDisabledModuleDeniesReadsAndRefusesWrites(t *testing.T) {
	t.Parallel()

	mod, err := dynconf.New(config.Module{Enabled: false, Driver: "file"}, dynconf.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	ctx := context.Background()

	if mod.IsEnabled(ctx, "nouveau_paiement") {
		t.Error("un module désactivé ne doit activer aucun drapeau")
	}
	if got := mod.GetSetting(ctx, "seuil"); got.Found {
		t.Error("un module désactivé ne doit trouver aucun réglage")
	}

	change := domain.Change{Kind: domain.KindFlag, Key: "nouveau_paiement", Value: "true"}
	if err := mod.Set(ctx, change); !errors.Is(err, dynconf.ErrDisabled) {
		t.Errorf("Set = %v, attendu ErrDisabled", err)
	}

	mod.Invalidate() // ne doit pas paniquer sur un module éteint
}
