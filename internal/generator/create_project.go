package generator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ProjectReport rend compte d'une génération réussie.
//
// Un type plutôt qu'un second retour : `CreateProject` en aurait alors trois, ce
// que la règle `arch-go` refuse — et elle a raison, c'est la leçon apprise cinq
// fois dans ce dépôt.
type ProjectReport struct {
	// Files est le nombre de fichiers recopiés.
	Files int
	// GitInitialised dit si le dépôt et ses crochets ont pu être posés.
	GitInitialised bool
}

// CreateProject déroule la génération, puis la VÉRIFIE.
//
// La vérification fait partie de la commande, elle n'est pas laissée à
// l'utilisateur : un projet généré qui ne compile pas doit le dire lui-même,
// pas attendre le premier `go build` de quelqu'un d'autre.
func CreateProject(ctx context.Context, p ProjectPlan) (ProjectReport, error) {
	files, err := TrackedFiles(ctx, p.Source)
	if err != nil {
		return ProjectReport{}, err
	}
	if copied := CopyProject(p, files); copied != nil {
		return ProjectReport{}, copied
	}

	remaining, err := VerifyNoTrace(p)
	if err != nil {
		return ProjectReport{}, err
	}
	if len(remaining) > 0 {
		return ProjectReport{}, fmt.Errorf(
			"le chemin du socle subsiste dans %d fichier(s) — le projet dépendrait en "+
				"silence d'un autre dépôt :\n  %s\n\nLes ajouter à la réécriture, ou les "+
				"déclarer dans CitesSocleByHistory — jamais les laisser passer",
			len(remaining), strings.Join(remaining, "\n  "))
	}

	if err := compile(ctx, p.Destination); err != nil {
		return ProjectReport{}, err
	}
	return ProjectReport{Files: len(files), GitInitialised: initGit(ctx, p.Destination) == nil}, nil
}

// compile éprouve le projet généré.
func compile(ctx context.Context, destination string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = destination
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("le projet généré ne compile pas — la génération est fautive, "+
			"pas le projet :\n%s", out)
	}
	return nil
}

// initGit pose le dépôt ET le chemin des crochets.
//
// Les deux vont ensemble : `git init` seul laisserait le crochet `commit-msg`
// inerte, donc un projet neuf sans son filet anti-accident.
//
// L'échec n'est pas bloquant — git peut être absent — mais il est RENDU, pas
// avalé : c'est l'appelant qui décide de le dire, et `ProjectReport` le porte.
func initGit(ctx context.Context, destination string) error {
	// Deux appels écrits en toutes lettres plutôt qu'une boucle sur des
	// arguments variables : `gosec` refuse le second, et il a raison de le
	// refuser — une commande dont les arguments viennent d'une variable est
	// exactement la forme qu'on ne veut pas relire sans réfléchir.
	steps := []*exec.Cmd{
		exec.CommandContext(ctx, "git", "init", "--quiet"),
		exec.CommandContext(ctx, "git", "config", "core.hooksPath", ".githooks"),
	}
	for _, cmd := range steps {
		cmd.Dir = destination
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dépôt git non initialisé: %w\n%s", err, out)
		}
	}
	return nil
}
