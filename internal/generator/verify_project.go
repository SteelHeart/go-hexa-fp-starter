package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyNoTrace refuse un projet qui porte encore le chemin du socle.
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
//
// Rend la liste des fichiers fautifs plutôt que de la journaliser : c'est
// l'appelant — la commande — qui décide comment la présenter. Un paquet de
// bibliothèque n'écrit pas sur la sortie d'erreur de son appelant.
func VerifyNoTrace(p ProjectPlan) ([]string, error) {
	var remaining []string

	err := filepath.WalkDir(p.Destination, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(p.Destination, path)
		if err != nil {
			return fmt.Errorf("chemin relatif de %s: %w", path, err)
		}
		if CitesSocleByHistory(filepath.ToSlash(relative)) {
			return nil
		}
		content, err := os.ReadFile(path) //nolint:gosec // chemin issu du parcours de la destination qu'on vient d'écrire
		if err != nil {
			return fmt.Errorf("relecture de %s: %w", relative, err)
		}
		if strings.Contains(string(content), p.SocleModule) {
			remaining = append(remaining, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("vérification du projet généré: %w", err)
	}
	return remaining, nil
}
