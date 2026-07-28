package main

import (
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRendreGabaritEcritUneAnatomieComplete : le module créé porte TOUS les
// dossiers de l'anatomie, et chaque fichier Go est formaté.
//
// # Pourquoi la complétude est vérifiée, et pas seulement la compilation
//
// `CLAUDE.md` le dit d'une phrase qui vaut pour ce test : *tout dossier manquant
// serait reproduit comme « pas nécessaire »*. Un gabarit qui oublierait
// `drivers/` produirait des modules sans pilote interchangeable, et personne ne
// s'en apercevrait avant le jour où il faut en changer.
//
// Le formatage est vérifié pour une raison mesurée : un gabarit ne peut pas être
// `gofmt`-propre par construction, puisque la largeur des identifiants dépend du
// nom du module. La première version produisait trois fichiers que `go fmt`
// réécrivait — donc une étape rouge dans le projet généré.
func TestRendreGabaritEcritUneAnatomieComplete(t *testing.T) {
	t.Parallel()

	p, err := planifierFeature("order_tracking", projetFactice(t))
	if err != nil {
		t.Fatalf("planification: %v", err)
	}
	if err := rendreGabarit(p); err != nil {
		t.Fatalf("rendu: %v", err)
	}

	attendus := []string{
		"module.go",
		"catalog.go",
		filepath.Join("domain", "errors.go"),
		filepath.Join("domain", "reference.go"),
		filepath.Join("domain", "record.go"),
		filepath.Join("ports", "ports.go"),
		filepath.Join("application", "create_record.go"),
		filepath.Join("drivers", "memory", "memory.go"),
	}
	for _, relatif := range attendus {
		if _, err := os.Stat(filepath.Join(p.destination, relatif)); err != nil {
			t.Errorf("l'anatomie est incomplète, %s manque: %v", relatif, err)
		}
	}

	for _, dossier := range []string{"domain/tests", "application/tests", "tests"} {
		entrees, err := os.ReadDir(filepath.Join(p.destination, filepath.FromSlash(dossier)))
		if err != nil || len(entrees) == 0 {
			t.Errorf("%s doit contenir des tests en boîte noire (%v)", dossier, err)
		}
	}

	verifierFormatage(t, p.destination)
}

// TestRendreGabaritReecritLeCheminDeModule : aucun fichier ne conserve le chemin
// de module du socle.
//
// C'est la faute qui rendrait le module inutilisable ailleurs, et elle est
// silencieuse : le projet compilerait, en tirant le code depuis le dépôt du
// socle plutôt que depuis le sien.
func TestRendreGabaritReecritLeCheminDeModule(t *testing.T) {
	t.Parallel()

	p, err := planifierFeature("billing", projetFactice(t))
	if err != nil {
		t.Fatalf("planification: %v", err)
	}
	if err := rendreGabarit(p); err != nil {
		t.Fatalf("rendu: %v", err)
	}

	module := lire(t, filepath.Join(p.destination, "module.go"))

	if !strings.Contains(module, "github.com/exemple/projet/internal/modules/billing/domain") {
		t.Error("les imports doivent porter le chemin de module du PROJET")
	}
	if strings.Contains(module, "go-hexa-fp-starter") {
		t.Error("le chemin de module du socle a survécu au rendu")
	}
	if !strings.Contains(module, "package billing") {
		t.Error("le paquet doit porter le nom dérivé du module")
	}
	if strings.Contains(module, "<no value>") {
		t.Error("une clé de gabarit n'a pas été résolue")
	}
}

// verifierFormatage échoue si un fichier Go n'est pas déjà `gofmt`-propre.
func verifierFormatage(t *testing.T, racine string) {
	t.Helper()

	err := filepath.Walk(racine, func(chemin string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(chemin, ".go") {
			return err
		}
		brut := lire(t, chemin)
		attendu, err := format.Source([]byte(brut))
		if err != nil {
			t.Errorf("%s n'est pas du Go valide: %v", chemin, err)
			return nil
		}
		if string(attendu) != brut {
			t.Errorf("%s n'est pas formaté — `go fmt` le réécrirait", chemin)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours: %v", err)
	}
}
