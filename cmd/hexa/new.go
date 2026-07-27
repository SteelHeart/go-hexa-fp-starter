package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// commandeNew crée un projet à partir du socle.
//
// Les étapes sont volontairement séparées et chacune REFUSE plutôt que de
// réparer : un générateur qui rattrape silencieusement une entrée douteuse
// produit un projet dont personne ne sait dans quel état il est.
func commandeNew(args []string) error {
	jeu := flag.NewFlagSet("new", flag.ContinueOnError)
	jeu.SetOutput(os.Stderr)
	module := jeu.String("module", "", "chemin de module Go du projet créé (obligatoire)")
	depuis := jeu.String("depuis", ".", "racine du socle à recopier")

	options, positionnels := separerArguments(args)
	if err := jeu.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positionnels) != 1 {
		return errors.New("usage : hexa new <destination> --module <chemin/de/module>")
	}

	prepare, err := planifier(positionnels[0], *module, *depuis)
	if err != nil {
		return err
	}
	return executer(context.Background(), prepare)
}

// plan porte les valeurs vérifiées d'une génération.
//
// Un type plutôt que quatre paramètres : la règle des deux retours vaut aussi
// pour les entrées, et quatre chaînes de suite s'inversent silencieusement.
type plan struct {
	source      string
	destination string
	moduleSocle string
	moduleCible string
}

// planifier vérifie TOUT avant d'écrire quoi que ce soit.
func planifier(destination, moduleCible, source string) (plan, error) {
	if moduleCible == "" {
		return plan{}, errors.New("--module est obligatoire : un projet sans chemin de module ne compile pas")
	}
	if !strings.Contains(moduleCible, "/") {
		return plan{}, fmt.Errorf(
			"--module=%q ne ressemble pas à un chemin de module (attendu : hôte/organisation/nom)", moduleCible)
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		return plan{}, fmt.Errorf("chemin du socle: %w", err)
	}
	moduleSocle, err := moduleDe(absSource)
	if err != nil {
		return plan{}, err
	}
	if moduleSocle == moduleCible {
		return plan{}, fmt.Errorf(
			"--module=%q est déjà le module du socle : rien à réécrire", moduleCible)
	}

	absDest, err := filepath.Abs(destination)
	if err != nil {
		return plan{}, fmt.Errorf("chemin de destination: %w", err)
	}
	if err := destinationLibre(absDest); err != nil {
		return plan{}, err
	}

	return plan{source: absSource, destination: absDest, moduleSocle: moduleSocle, moduleCible: moduleCible}, nil
}

// destinationLibre refuse d'écrire dans un répertoire non vide.
//
// Deny par défaut : écraser des fichiers existants est irréversible, et un
// générateur n'a aucun moyen de savoir ce qui comptait pour l'utilisateur.
func destinationLibre(chemin string) error {
	entrees, err := os.ReadDir(chemin)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return fmt.Errorf("lecture de la destination: %w", err)
	case len(entrees) > 0:
		return fmt.Errorf("%s n'est pas vide — choisir un répertoire neuf", chemin)
	}
	return nil
}

// moduleDe lit le chemin de module déclaré par le socle.
func moduleDe(racine string) (string, error) {
	// Racine fournie par l'utilisateur, et c'est voulu : il désigne SON socle.
	//nolint:gosec // racine désignée par l'appelant, nom de fichier fixe
	brut, err := os.ReadFile(filepath.Join(racine, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("%s n'a pas de go.mod — est-ce bien la racine du socle ? %w", racine, err)
	}
	for ligne := range strings.SplitSeq(string(brut), "\n") {
		if reste, trouve := strings.CutPrefix(strings.TrimSpace(ligne), "module "); trouve {
			return strings.TrimSpace(reste), nil
		}
	}
	return "", fmt.Errorf("aucune directive `module` dans %s/go.mod", racine)
}

// executer déroule la génération, puis la VÉRIFIE.
//
// La vérification fait partie de la commande, elle n'est pas laissée à
// l'utilisateur : un projet généré qui ne compile pas doit le dire lui-même,
// pas attendre le premier `go build` de quelqu'un d'autre.
func executer(ctx context.Context, p plan) error {
	fichiers, err := fichiersSuivis(ctx, p.source)
	if err != nil {
		return err
	}
	if err := copier(p, fichiers); err != nil {
		return err
	}
	if err := verifierAucuneTrace(p); err != nil {
		return err
	}
	if err := compiler(ctx, p.destination); err != nil {
		return err
	}
	if err := initialiserGit(ctx, p.destination); err != nil {
		return err
	}

	fmt.Printf("Projet créé dans %s\n", p.destination)
	fmt.Printf("  module   %s\n", p.moduleCible)
	fmt.Printf("  fichiers %d\n\n", len(fichiers))
	fmt.Print(`Ensuite :
  cd <destination>
  task init          # .env et outillage
  task check         # la barrière qualité, identique à la CI

Le module de démonstration ` + "`internal/modules/user_registration`" + ` est la
TRANCHE DE RÉFÉRENCE : sa forme est celle à copier pour écrire un module métier.
Le supprimer exige de retirer son montage de cmd/server.
`)
	return nil
}

// compiler éprouve le projet généré.
func compiler(ctx context.Context, destination string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = destination
	if sortie, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("le projet généré ne compile pas — la génération est fautive, "+
			"pas le projet :\n%s", sortie)
	}
	return nil
}

// initialiserGit pose le dépôt ET le chemin des crochets.
//
// Les deux vont ensemble : `git init` seul laisserait le crochet `commit-msg`
// inerte, donc un projet neuf sans son filet anti-accident. L'échec n'est pas
// bloquant — git peut être absent — mais il se dit.
func initialiserGit(ctx context.Context, destination string) error {
	// Deux appels écrits en toutes lettres plutôt qu'une boucle sur des
	// arguments variables : `gosec` refuse le second, et il a raison de le
	// refuser — une commande dont les arguments viennent d'une variable est
	// exactement la forme qu'on ne veut pas relire sans réfléchir.
	commandes := []*exec.Cmd{
		exec.CommandContext(ctx, "git", "init", "--quiet"),
		exec.CommandContext(ctx, "git", "config", "core.hooksPath", ".githooks"),
	}
	for _, cmd := range commandes {
		cmd.Dir = destination
		if sortie, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr,
				"hexa: dépôt git non initialisé (%v) — lancer `git init` et "+
					"`git config core.hooksPath .githooks` à la main\n%s", err, sortie)
			return nil
		}
	}
	return nil
}

// separerArguments range les options d'un côté, la destination de l'autre.
//
// Le paquet `flag` s'arrête au PREMIER argument non-option : sans ce tri,
// `hexa new ./projet --module x` ignorerait `--module` en silence, et la
// commande échouerait en accusant l'absence d'une option pourtant écrite.
//
// Imposer l'ordre `--module x ./projet` serait une friction gratuite dans un
// outil dont la raison d'être est d'en supprimer.
func separerArguments(args []string) (options, positionnels []string) {
	aValeur := map[string]bool{"-module": true, "--module": true, "-depuis": true, "--depuis": true}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case !strings.HasPrefix(arg, "-"):
			positionnels = append(positionnels, arg)
		case aValeur[arg] && i+1 < len(args):
			// Forme `--module x` : la valeur suit, elle n'est pas positionnelle.
			options = append(options, arg, args[i+1])
			i++
		default:
			// Forme `--module=x`, ou option inconnue que `flag` refusera lui-même.
			options = append(options, arg)
		}
	}
	return options, positionnels
}
