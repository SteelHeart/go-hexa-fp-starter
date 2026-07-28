package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// gabaritFeature porte l'anatomie d'un module métier.
//
// Embarquée plutôt que lue sur le disque : le binaire `hexa` doit fonctionner
// depuis n'importe où, y compris installé hors du dépôt.
//
// Les fichiers portent le suffixe `.tmpl` pour une raison mécanique : nommés
// `.go`, ils seraient compilés avec le reste du paquet, et un gabarit ne compile
// pas — il contient `{{.Module}}`.
//
//go:embed all:gabarit/feature
var gabaritFeature embed.FS

// racineGabaritFeature est le préfixe à retirer des chemins embarqués.
const racineGabaritFeature = "gabarit/feature"

// nomDeModuleValide contraint le nom d'un module métier.
//
// `snake_case` strict, et c'est une contrainte de forme, pas de goût : le nom
// devient un nom de RÉPERTOIRE, une clé de `config/modules.yaml`, et — dépouillé
// de ses tirets bas — un nom de PAQUET Go. Accepter `Mon-Module` produirait un
// paquet `MonModule`, que `revive` refuse, dans un projet dont la barrière
// échouerait dès la première commande.
var nomDeModuleValide = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// commandeFeature crée un module métier dans un projet existant.
func commandeFeature(args []string) error {
	jeu := flag.NewFlagSet("make:feature", flag.ContinueOnError)
	jeu.SetOutput(os.Stderr)
	dans := jeu.String("dans", ".", "racine du projet où créer le module")

	options, positionnels := separerArguments(args, flagsAValeur(jeu))
	if err := jeu.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positionnels) != 1 {
		return errors.New("usage : hexa make:feature <nom_du_module>")
	}

	prepare, err := planifierFeature(positionnels[0], *dans)
	if err != nil {
		return err
	}
	return executerFeature(context.Background(), prepare)
}

// planFeature porte les valeurs vérifiées d'une création de module.
type planFeature struct {
	racine      string
	destination string
	// Module est le chemin de module Go du PROJET, pas du module métier.
	Module string
	// Dir est le nom du répertoire, en snake_case : `order_tracking`.
	Dir string
	// Package est le nom du paquet Go, sans tiret bas : `ordertracking`.
	Package string
	// ancrage porte le point d'insertion de la règle d'étanchéité d'arch-go.
	//
	// Repéré pendant la PLANIFICATION, écrit pendant l'exécution : sans cela, un
	// arch-go.yml inattendu ferait échouer la commande après avoir déjà créé le
	// module, et laisserait un module hors garde sur le disque.
	ancrage ancrageIsolation
}

// planifierFeature vérifie TOUT avant d'écrire quoi que ce soit.
func planifierFeature(nom, racine string) (planFeature, error) {
	if !nomDeModuleValide.MatchString(nom) {
		return planFeature{}, fmt.Errorf(
			"nom de module %q invalide — attendu du snake_case : `billing`, `order_tracking`", nom)
	}

	absRacine, err := filepath.Abs(racine)
	if err != nil {
		return planFeature{}, fmt.Errorf("chemin du projet: %w", err)
	}
	module, err := moduleDe(absRacine)
	if err != nil {
		return planFeature{}, err
	}

	destination := filepath.Join(absRacine, "internal", "modules", nom)
	if occupe := destinationLibre(destination); occupe != nil {
		return planFeature{}, fmt.Errorf("le module existe déjà: %w", occupe)
	}

	ancrage, err := trouverAncrage(absRacine, nom)
	if err != nil {
		return planFeature{}, err
	}

	return planFeature{
		racine:      absRacine,
		destination: destination,
		Module:      module,
		Dir:         nom,
		Package:     strings.ReplaceAll(nom, "_", ""),
		ancrage:     ancrage,
	}, nil
}

// executerFeature rend le gabarit, durcit l'architecture, puis VÉRIFIE.
func executerFeature(ctx context.Context, p planFeature) error {
	if err := rendreGabarit(p); err != nil {
		return err
	}
	if err := declarerIsolation(p.ancrage, p.Dir); err != nil {
		return err
	}
	if err := eprouver(ctx, p); err != nil {
		return err
	}
	annoncerCablage(p)
	return nil
}

// rendreGabarit écrit l'arborescence du module.
func rendreGabarit(p planFeature) error {
	parcours := fs.WalkDir(gabaritFeature, racineGabaritFeature,
		func(chemin string, entree fs.DirEntry, err error) error {
			if err != nil || entree.IsDir() {
				return err
			}
			relatif := strings.TrimPrefix(strings.TrimPrefix(chemin, racineGabaritFeature), "/")
			cible := filepath.Join(p.destination, strings.TrimSuffix(relatif, ".tmpl"))
			return rendreUn(gabaritFeature, chemin, cible, p)
		})
	if parcours != nil {
		return fmt.Errorf("rendu du gabarit de module: %w", parcours)
	}
	return nil
}

// rendreUn écrit un fichier du gabarit, formaté.
func rendreUn(source fs.FS, chemin, cible string, p planFeature) error {
	rendu, err := rendre(source, chemin, p)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cible), 0o750); err != nil {
		return fmt.Errorf("création de %s: %w", filepath.Dir(cible), err)
	}
	if err := os.WriteFile(cible, rendu, 0o600); err != nil {
		return fmt.Errorf("écriture de %s: %w", cible, err)
	}
	return nil
}

// rendre applique le gabarit, puis FORMATE le résultat s'il est en Go.
//
// # Pourquoi le formatage est ici et non dans les gabarits
//
// Un gabarit ne peut pas être `gofmt`-propre par construction : la largeur des
// identifiants change avec le nom du module, donc tout alignement écrit à la main
// est faux pour tous les noms sauf un. Mesuré — la première version produisait
// trois fichiers que `go fmt` réécrivait, et l'étape `fmt` de la barrière du
// projet généré les signalait.
//
// `format.Source` supprime la classe entière : aucun futur remaniement d'un
// gabarit ne peut plus produire du code mal formaté. Et un gabarit devenu
// syntaxiquement faux ne s'écrit plus en silence — il refuse ici, avec sa
// position, au lieu d'échouer plus tard sur un `go build` du projet.
func rendre(source fs.FS, chemin string, p planFeature) ([]byte, error) {
	brut, err := fs.ReadFile(source, chemin)
	if err != nil {
		return nil, fmt.Errorf("lecture du gabarit %s: %w", chemin, err)
	}
	// `missingkey=error` : un gabarit qui référencerait une clé absente écrirait
	// `<no value>` dans du code Go. Mieux vaut refuser bruyamment.
	modele, err := template.New(chemin).Option("missingkey=error").Parse(string(brut))
	if err != nil {
		return nil, fmt.Errorf("gabarit %s illisible: %w", chemin, err)
	}

	var tampon bytes.Buffer
	if rendu := modele.Execute(&tampon, p); rendu != nil {
		return nil, fmt.Errorf("rendu de %s: %w", chemin, rendu)
	}
	if !strings.HasSuffix(chemin, ".go.tmpl") {
		return tampon.Bytes(), nil
	}

	formate, err := format.Source(tampon.Bytes())
	if err != nil {
		return nil, fmt.Errorf("le gabarit %s ne produit pas du Go valide: %w", chemin, err)
	}
	return formate, nil
}

// eprouver compile le projet ENTIER et exécute TOUS ses tests.
//
// La vérification fait partie de la commande, elle n'est pas laissée à
// l'utilisateur : un module généré qui ne compile pas doit le dire lui-même.
//
// Le projet entier plutôt que les seuls paquets du module neuf, pour deux
// raisons. La première est de fond : un module qui casse le reste du projet est
// un module fautif, et ne le constater qu'au `task check` suivant ferait porter
// le doute sur la mauvaise modification. La seconde est mécanique — `gosec`
// refuse à juste titre une commande dont un argument est une variable, et une
// dérogation motivée coûterait plus cher que la vérification plus large.
func eprouver(ctx context.Context, p planFeature) error {
	etapes := []*exec.Cmd{
		exec.CommandContext(ctx, "go", "build", "./..."),
		exec.CommandContext(ctx, "go", "test", "./..."),
	}
	for _, cmd := range etapes {
		cmd.Dir = p.racine
		if sortie, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("le module généré ne passe pas `%s` — la génération est "+
				"fautive, pas le module :\n%s", strings.Join(cmd.Args, " "), sortie)
		}
	}
	return nil
}

// annoncerCablage imprime les lignes à ajouter, et dit pourquoi elles ne sont
// pas ajoutées automatiquement.
func annoncerCablage(p planFeature) {
	fmt.Printf("Module %s créé dans %s\n\n", p.Dir, p.destination)
	fmt.Printf(`Trois lignes restent à écrire, et elles ne sont PAS écrites pour vous :
monter un module est une décision par binaire (ADR 014), pas une conséquence de
sa création. Un générateur qui câblerait tout seul produirait des modules montés
que personne n'a choisi de monter.

1. internal/modules/catalog.go — rendre le module DÉCLARABLE

       %s.Catalog(),

2. config/modules.yaml — l'activer

       %s:
         driver: memory

3. cmd/server/main.go — le monter

       module, err := %s.New(cfg.Modules.DriverOf(%s.Name), %s.Deps{
           GenerateID: ...,
           Now:        %s.SystemClock(),
       })

Puis : task check
`, p.Package, p.Dir, p.Package, p.Package, p.Package, p.Package)
}
