package tests

import (
	"context"
	"testing"
)

// TestUnknownFlagIsDenied : c'est LA garantie du module. Un drapeau qui
// s'activerait sans qu'on l'ait posé mettrait en production la fonctionnalité
// incomplète qu'il est justement censé masquer (ADR 007, tronc unique).
func TestUnknownFlagIsDenied(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"flags": map[string]any{"connu": true},
	})
	ctx := context.Background()

	if !mod.IsEnabled(ctx, "connu") {
		t.Error("un drapeau déclaré actif doit être actif")
	}
	if mod.IsEnabled(ctx, "inconnu") {
		t.Error("un drapeau jamais déclaré doit être inactif")
	}
}
