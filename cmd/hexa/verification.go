package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// verifierAucuneTrace refuse un projet qui porte encore le chemin du socle.
//
// # Pourquoi cette vérification est DANS la commande
//
// Une réécriture partielle produit un projet qui compile — Go résout l'import
// vers le socle d'origine s'il est accessible — mais qui dépend en silence d'un
// autre dépôt. Le symptôme arrive des semaines plus tard, sous la forme d'un
// paquet qu'on ne trouve plus.
//
// C'est le sens de l'ADR 013 appliqué au générateur : il est livré avec le cas
// qui le fait échouer, et ce cas est vérifié à CHAQUE exécution plutôt qu'une
// fois en test.
func verifierAucuneTrace(p plan) error {
	var restants []string

	err := filepath.WalkDir(p.destination, func(chemin string, entree os.DirEntry, err error) error {
		if err != nil || entree.IsDir() {
			return err
		}
		relatif, err := filepath.Rel(p.destination, chemin)
		if err != nil {
			return fmt.Errorf("chemin relatif de %s: %w", chemin, err)
		}
		if citeLeSocleParHistoire(filepath.ToSlash(relatif)) {
			return nil
		}
		contenu, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours de la destination qu'on vient d'écrire
		if err != nil {
			return fmt.Errorf("relecture de %s: %w", relatif, err)
		}
		if strings.Contains(string(contenu), p.moduleSocle) {
			restants = append(restants, filepath.ToSlash(relatif))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("vérification du projet généré: %w", err)
	}

	if len(restants) > 0 {
		// Le détail va sur stderr, le message d'erreur reste une phrase : un
		// `error` qui porte plusieurs lignes se relit mal une fois enveloppé.
		fmt.Fprintf(os.Stderr, "hexa: le chemin du socle subsiste dans :\n  %s\n\n"+
			"Ajouter ces fichiers à la réécriture, ou les déclarer dans "+
			"citeLeSocleParHistoire — jamais les laisser passer.\n",
			strings.Join(restants, "\n  "))
		return fmt.Errorf("le chemin du socle subsiste dans %d fichier(s) : le projet "+
			"dépendrait en silence d'un autre dépôt", len(restants))
	}
	return nil
}
