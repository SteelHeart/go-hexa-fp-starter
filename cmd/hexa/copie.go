package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// fichiersSuivis énumère ce qui appartient au socle.
//
// `git ls-files` plutôt qu'un parcours du disque, et ce n'est pas un raccourci :
// c'est la seule définition qui ne se périme pas. Elle écarte d'office `.git/`,
// `bin/`, `.env`, `coverage.out` et tout ce que `.gitignore` désignera demain —
// alors qu'une liste d'exclusions écrite ici aurait divergé au premier ajout.
//
// Corollaire assumé : un fichier non suivi n'est pas recopié. C'est le bon
// comportement — un fichier que le socle ne versionne pas ne fait pas partie du
// socle.
func fichiersSuivis(ctx context.Context, racine string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "-z")
	cmd.Dir = racine
	sortie, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s n'est pas un dépôt git — `hexa new` recopie les fichiers "+
			"SUIVIS, ce qui exige un dépôt: %w", racine, err)
	}

	var fichiers []string
	for nom := range strings.SplitSeq(string(sortie), "\x00") {
		if nom != "" {
			fichiers = append(fichiers, nom)
		}
	}
	if len(fichiers) == 0 {
		return nil, fmt.Errorf("aucun fichier suivi dans %s : il n'y aurait rien à copier", racine)
	}
	return fichiers, nil
}

// copier recopie chaque fichier en réécrivant le chemin de module au passage.
//
// La réécriture se fait pendant la copie, jamais après : un passage séparé sur
// l'arborescence oublierait les fichiers ajoutés entre-temps, et surtout il
// laisserait un état intermédiaire où le projet ne compile pas.
func copier(p plan, fichiers []string) error {
	for _, relatif := range fichiers {
		if err := copierUn(p, relatif); err != nil {
			return err
		}
	}
	return nil
}

// copierUn traite un fichier, en PRÉSERVANT ses permissions.
//
// Le bit exécutable n'est pas cosmétique ici : `.githooks/commit-msg` et les
// gardes de `tools/` ne s'exécutent pas sans lui. Ce dépôt a déjà payé cette
// erreur — ses deux crochets étaient versionnés en 100644, donc git les ignorait
// partout, sur toutes les machines, sans que rien ne le signale.
func copierUn(p plan, relatif string) error {
	source := filepath.Join(p.source, relatif)
	cible := filepath.Join(p.destination, relatif)

	// `git ls-files` ne rend jamais de chemin hors du dépôt, mais l'affirmer ne
	// suffit pas : c'est le genre d'invariant qui tient jusqu'au jour où
	// quelqu'un appelle cette fonction autrement. Le refus est explicite.
	if !strings.HasPrefix(cible, p.destination+string(filepath.Separator)) {
		return fmt.Errorf("%s sort de la destination : chemin refusé", relatif)
	}

	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("lecture de %s: %w", relatif, err)
	}
	// Un lien symbolique recopié en fichier changerait le sens de l'arborescence.
	// Le socle n'en contient aucun ; le jour où il en contiendra, il faudra en
	// décider explicitement plutôt que de découvrir le résultat.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s est un lien symbolique : cas non traité, à trancher avant de le versionner", relatif)
	}

	// Le chemin vient de `git ls-files` sur la racine fournie, jamais d'une
	// entrée arbitraire : c'est la liste des fichiers du socle lui-même.
	contenu, err := os.ReadFile(source) //nolint:gosec // chemin issu de git ls-files sur la racine du socle
	if err != nil {
		return fmt.Errorf("lecture de %s: %w", relatif, err)
	}
	if !citeLeSocleParHistoire(relatif) {
		contenu = []byte(strings.ReplaceAll(string(contenu), p.moduleSocle, p.moduleCible))
	}

	if err := os.MkdirAll(filepath.Dir(cible), 0o750); err != nil {
		return fmt.Errorf("création du répertoire de %s: %w", relatif, err)
	}
	//nolint:gosec // cible vérifiée plus haut comme interne à la destination
	if err := os.WriteFile(cible, contenu, info.Mode().Perm()); err != nil {
		return fmt.Errorf("écriture de %s: %w", relatif, err)
	}
	return nil
}

// citeLeSocleParHistoire nomme les fichiers dont les occurrences du chemin de
// module sont des LIENS vers l'historique du socle, pas des imports.
//
// Les réécrire ferait pointer l'historique du socle vers un dépôt qui ne l'a
// jamais porté : des liens vers des PR et des issues qui n'existent pas.
//
// La liste est identique à celle de `rename:verify` dans le Taskfile, et pour la
// même raison. Elle est ÉNUMÉRÉE, jamais devinée par motif : une exception qui
// s'élargit toute seule finit par couvrir un vrai import.
func citeLeSocleParHistoire(relatif string) bool {
	const separateur = "/"
	switch {
	case relatif == "CLAUDE.md":
		return true
	case relatif == "documentation/process/REPRISE.md":
		return true
	case strings.HasPrefix(relatif, "documentation/adr"+separateur):
		// Chaque ADR cite son issue d'origine en en-tête.
		return true
	default:
		return false
	}
}
