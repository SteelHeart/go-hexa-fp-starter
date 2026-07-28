// Package generator porte tout ce que fait la commande `hexa`, moins
// l'aiguillage.
//
// # Pourquoi cette logique n'est pas dans `cmd/hexa`
//
// Elle y était, et c'était une faute mesurable : `cmd/hexa` est un
// `package main`, et **Go interdit d'importer un paquet `main`**. Ses tests ne
// pouvaient donc être ni en boîte noire, ni dans `{paquet}/tests/` — les deux
// exigences de `rules/tests.md`. Dix fichiers de test s'étaient accumulés à la
// racine du paquet, en dehors des deux emplacements que la règle prévoit (#96).
//
// Le déplacement corrige trois choses d'un coup :
//
//   - les tests deviennent possibles **par l'API publique**, dans `tests/` ;
//   - `covergate` les COMPTE enfin, `cmd/` étant hors du périmètre unitaire ;
//   - `cmd/hexa/main.go` redevient ce qu'un composition root doit être — une
//     coquille mince qui déclare des drapeaux et aiguille (ADR 004).
//
// # Ce paquet ne connaît RIEN de l'application
//
// Il manipule des fichiers, pas des modules. Il n'importe ni `internal/core`,
// ni `internal/modules`, ni `internal/infrastructure` — vérifié par `arch-go`.
// C'est ce qui le rendra extractible le jour où le socle deviendra une
// bibliothèque importable (ADR 015).
//
// # Carte des fichiers
//
//	arguments.go        tri des arguments avant `flag`
//	plan_project.go     PlanProject — vérifie TOUT avant d'écrire
//	create_project.go   CreateProject — copie, vérifie, compile, initialise git
//	copy_project.go     TrackedFiles, CopyProject
//	verify_project.go   VerifyNoTrace — aucun reste du chemin du socle
//	plan_feature.go     PlanFeature
//	create_feature.go   CreateFeature, RenderFeature
//	isolation.go        la règle d'étanchéité `arch-go` du module créé
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EmptyDestination refuse d'écrire dans un répertoire non vide.
//
// Deny par défaut : écraser des fichiers existants est irréversible, et un
// générateur n'a aucun moyen de savoir ce qui comptait pour l'utilisateur.
func EmptyDestination(path string) error {
	entries, err := os.ReadDir(path)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("lecture de la destination: %w", err)
	case len(entries) > 0:
		return fmt.Errorf("%s n'est pas vide — choisir un répertoire neuf", path)
	}
	return nil
}

// ModulePathOf lit le chemin de module déclaré par un projet Go.
func ModulePathOf(root string) (string, error) {
	// Racine fournie par l'appelant, et c'est voulu : il désigne SON projet.
	//nolint:gosec // racine désignée par l'appelant, nom de fichier fixe
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("%s n'a pas de go.mod — est-ce bien la racine du projet ? %w", root, err)
	}
	for line := range strings.SplitSeq(string(raw), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "module "); found {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("aucune directive `module` dans %s/go.mod", root)
}
