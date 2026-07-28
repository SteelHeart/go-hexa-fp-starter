package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlanifierRefuseAvantDEcrire : toutes les vérifications ont lieu AVANT la
// première écriture.
//
// Un générateur qui découvre un problème au milieu de la copie laisse une
// destination à moitié écrite, dont personne ne sait si elle est utilisable. Le
// refus doit donc être complet et précoce.
//
// Chaque cas est un refus, et chaque refus doit NOMMER ce qui cloche : un
// message qui dit seulement « erreur » fait chercher au hasard.
func TestPlanifierRefuseAvantDEcrire(t *testing.T) {
	t.Parallel()

	socle := t.TempDir()
	if err := os.WriteFile(filepath.Join(socle, "go.mod"),
		[]byte("module github.com/exemple/socle\n\ngo 1.25.12\n"), 0o600); err != nil {
		t.Fatalf("préparation du socle: %v", err)
	}

	occupe := t.TempDir()
	if err := os.WriteFile(filepath.Join(occupe, "deja-la.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("préparation de la destination occupée: %v", err)
	}

	cas := map[string]struct {
		destination string
		module      string
		source      string
		motif       string
	}{
		"module absent": {
			destination: filepath.Join(t.TempDir(), "neuf"),
			module:      "",
			source:      socle,
			motif:       "obligatoire",
		},
		"module sans barre oblique": {
			destination: filepath.Join(t.TempDir(), "neuf"),
			module:      "facturation",
			source:      socle,
			motif:       "chemin de module",
		},
		"module identique au socle": {
			destination: filepath.Join(t.TempDir(), "neuf"),
			module:      "github.com/exemple/socle",
			source:      socle,
			motif:       "déjà le module du socle",
		},
		"socle sans go.mod": {
			destination: filepath.Join(t.TempDir(), "neuf"),
			module:      "github.com/exemple/cible",
			source:      t.TempDir(),
			motif:       "go.mod",
		},
		"destination non vide": {
			destination: occupe,
			module:      "github.com/exemple/cible",
			source:      socle,
			motif:       "n'est pas vide",
		},
	}

	for nom, tc := range cas {
		t.Run(nom, func(t *testing.T) {
			t.Parallel()
			_, err := planifier(tc.destination, tc.module, tc.source)
			if err == nil {
				t.Fatal("un plan invalide doit être refusé AVANT toute écriture")
			}
			if !strings.Contains(err.Error(), tc.motif) {
				t.Errorf("message = %q, il doit contenir %q pour être actionnable", err, tc.motif)
			}
		})
	}
}
