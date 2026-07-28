package generator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TrackedFiles énumère ce qui appartient au socle.
//
// `git ls-files` plutôt qu'un parcours du disque, et ce n'est pas un raccourci :
// c'est la seule définition qui ne se périme pas. Elle écarte d'office `.git/`,
// `bin/`, `.env`, `coverage.out` et tout ce que `.gitignore` désignera demain —
// alors qu'une liste d'exclusions écrite ici aurait divergé au premier ajout.
//
// Corollaire assumé : un fichier non suivi n'est pas recopié. C'est le bon
// comportement — un fichier que le socle ne versionne pas ne fait pas partie du
// socle.
func TrackedFiles(ctx context.Context, root string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s n'est pas un dépôt git — `hexa new` recopie les fichiers "+
			"SUIVIS, ce qui exige un dépôt: %w", root, err)
	}

	var files []string
	for name := range strings.SplitSeq(string(out), "\x00") {
		if name != "" {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("aucun fichier suivi dans %s : il n'y aurait rien à copier", root)
	}
	return files, nil
}

// CopyProject recopie chaque fichier en réécrivant le chemin de module au passage.
//
// La réécriture se fait pendant la copie, jamais après : un passage séparé sur
// l'arborescence oublierait les fichiers ajoutés entre-temps, et surtout il
// laisserait un état intermédiaire où le projet ne compile pas.
func CopyProject(p ProjectPlan, files []string) error {
	for _, relative := range files {
		if err := copyOne(p, relative); err != nil {
			return err
		}
	}
	return nil
}

// copyOne traite un fichier, en PRÉSERVANT ses permissions.
//
// Le bit exécutable n'est pas cosmétique ici : `.githooks/commit-msg` et les
// gardes de `tools/` ne s'exécutent pas sans lui. Ce dépôt a déjà payé cette
// erreur — ses deux crochets étaient versionnés en 100644, donc git les ignorait
// partout, sur toutes les machines, sans que rien ne le signale.
func copyOne(p ProjectPlan, relative string) error {
	source := filepath.Join(p.Source, relative)
	target := filepath.Join(p.Destination, relative)

	// `git ls-files` ne rend jamais de chemin hors du dépôt, mais l'affirmer ne
	// suffit pas : c'est le genre d'invariant qui tient jusqu'au jour où
	// quelqu'un appelle cette fonction autrement. Le refus est explicite.
	if !strings.HasPrefix(target, p.Destination+string(filepath.Separator)) {
		return fmt.Errorf("%s sort de la destination : chemin refusé", relative)
	}

	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("lecture de %s: %w", relative, err)
	}
	// Un lien symbolique recopié en fichier changerait le sens de l'arborescence.
	// Le socle n'en contient aucun ; le jour où il en contiendra, il faudra en
	// décider explicitement plutôt que de découvrir le résultat.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s est un lien symbolique : cas non traité, à trancher avant de le versionner", relative)
	}

	// Le chemin vient de `git ls-files` sur la racine fournie, jamais d'une
	// entrée arbitraire : c'est la liste des fichiers du socle lui-même.
	content, err := os.ReadFile(source) //nolint:gosec // chemin issu de git ls-files sur la racine du socle
	if err != nil {
		return fmt.Errorf("lecture de %s: %w", relative, err)
	}
	if !CitesSocleByHistory(relative) {
		content = []byte(strings.ReplaceAll(string(content), p.SocleModule, p.TargetModule))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("création du répertoire de %s: %w", relative, err)
	}
	//nolint:gosec // cible vérifiée plus haut comme interne à la destination
	if err := os.WriteFile(target, content, info.Mode().Perm()); err != nil {
		return fmt.Errorf("écriture de %s: %w", relative, err)
	}
	return nil
}

// CitesSocleByHistory nomme les fichiers dont les occurrences du chemin de
// module sont des LIENS vers l'historique du socle, pas des imports.
//
// Les réécrire ferait pointer l'historique du socle vers un dépôt qui ne l'a
// jamais porté : des liens vers des PR et des issues qui n'existent pas.
//
// La liste est identique à celle de `rename:verify` dans le Taskfile, et pour la
// même raison. Elle est ÉNUMÉRÉE, jamais devinée par motif : une exception qui
// s'élargit toute seule finit par couvrir un vrai import.
func CitesSocleByHistory(relative string) bool {
	const separator = "/"
	switch {
	case relative == "CLAUDE.md":
		return true
	case relative == "documentation/process/REPRISE.md":
		return true
	case strings.HasPrefix(relative, "documentation/adr"+separator):
		// Chaque ADR cite son issue d'origine en en-tête.
		return true
	default:
		return false
	}
}
