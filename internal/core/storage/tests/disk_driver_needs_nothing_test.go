package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage"
)

// TestDiskDriverNeedsNothing verrouille la promesse de l'ADR 012 : le pilote par
// défaut ne réclame ni base, ni bucket, ni client de fournisseur.
//
// Le répertoire de base est créé au démarrage plutôt qu'au premier téléversement :
// découvrir un magasin inutilisable quand un utilisateur envoie son premier
// fichier serait le pire moment.
func TestDiskDriverNeedsNothing(t *testing.T) {
	t.Parallel()

	mod, err := storage.New(config.Module{
		Enabled: true,
		Driver:  "disk",
		Options: map[string]any{"base_dir": t.TempDir()},
	}, storage.Deps{})
	if err != nil {
		t.Fatalf("le pilote par défaut ne doit réclamer aucune dépendance: %v", err)
	}

	if _, err := mod.Put(context.Background(), object("premier.txt", "contenu")); err != nil {
		t.Errorf("Put sur le pilote disk: %v", err)
	}
}
