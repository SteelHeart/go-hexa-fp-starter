package tests

import (
	"context"
	"testing"
)

// TestMissingSettingIsDistinctFromEmpty : « absent » et « présent et vide » sont
// deux situations différentes.
//
// Les confondre ferait appliquer une valeur par défaut à la place d'une décision
// d'exploitation délibérée — vider une bannière pour la faire disparaître la
// ferait réapparaître avec son texte d'origine.
func TestMissingSettingIsDistinctFromEmpty(t *testing.T) {
	t.Parallel()

	mod := newFileModule(t, map[string]any{
		"settings": map[string]any{"banniere": ""},
	})
	ctx := context.Background()

	vide := mod.GetSetting(ctx, "banniere")
	if !vide.Found {
		t.Error("un réglage déclaré vide doit être trouvé")
	}
	if vide.Value != "" {
		t.Errorf("valeur = %q, attendu vide", vide.Value)
	}

	if absent := mod.GetSetting(ctx, "jamais_pose"); absent.Found {
		t.Error("un réglage jamais posé ne doit pas être trouvé")
	}
}
