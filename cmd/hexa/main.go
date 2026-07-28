// Command hexa est l'outil en ligne de commande du socle.
//
// # Une coquille mince, et rien d'autre
//
// Ce fichier déclare des drapeaux, aiguille vers une sous-commande, et imprime.
// Toute la logique vit dans `internal/generator`, pour une raison mécanique
// autant qu'architecturale : Go interdit d'importer un paquet `main`, donc du
// code posé ici ne peut être testé ni en boîte noire, ni depuis `{paquet}/tests/`
// — les deux emplacements que `rules/tests.md` prévoit (#96).
//
// # Ce que fait `hexa` aujourd'hui, et ce qu'il fera
//
// `hexa new` recopie le socle et réécrit son chemin de module. C'est un
// GABARIT, pas une bibliothèque : le projet généré POSSÈDE tout le code, et un
// correctif du socle ne s'y propage pas.
//
// C'est un choix de séquencement, pas une cible (ADR 015). Aucun paquet du
// socle n'est importable — tout vit sous `internal/` — et personne n'a jamais
// construit d'application dessus. Décider maintenant ce qui devient public se
// ferait donc au jugé. `hexa new` est la seule façon de produire l'application
// dont la liste d'imports MESURERA cette frontière.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// version est injectée à la compilation, comme pour les deux autres binaires.
var version = "dev"

const usage = `hexa — outil du socle hexagonal

  hexa new <destination> --module <chemin/de/module>   crée un projet
  hexa make:feature <nom_du_module>                    crée un module métier
  hexa version                                          affiche la version

Exemples :
  hexa new ./mon-projet --module github.com/impactone/facturation
  hexa make:feature order_tracking

Options de « new » :
  --module   OBLIGATOIRE. Le chemin de module Go du projet créé.
  --depuis   Racine du socle à recopier (défaut : le répertoire courant).
             Ce doit être un dépôt git : ce sont les fichiers SUIVIS qui
             définissent le socle, ce qui écarte .git/, bin/ et .env.

Options de « make:feature » :
  --dans     Racine du projet où créer le module (défaut : le répertoire
             courant). Le nom est en snake_case : il devient un répertoire, une
             clé de config/modules.yaml, et — sans ses tirets bas — un paquet Go.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		// Sur stderr, et sans détruire la destination : un échec se diagnostique
		// en regardant ce qui a été écrit, pas en devinant ce qui manque.
		fmt.Fprintf(os.Stderr, "hexa: %v\n", err)
		os.Exit(1)
	}
}

// run aiguille vers la sous-commande.
func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "new":
		return commandNew(args[1:])
	case "make:feature":
		return commandFeature(args[1:])
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("sous-commande inconnue %q — `hexa help` liste ce qui existe", args[0])
	}
}

// commandNew crée un projet à partir du socle.
func commandNew(args []string) error {
	set := flag.NewFlagSet("new", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	module := set.String("module", "", "chemin de module Go du projet créé (obligatoire)")
	from := set.String("depuis", ".", "racine du socle à recopier")

	options, positional := generator.SplitArguments(args, generator.FlagsWithValue(set))
	if err := set.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positional) != 1 {
		return errors.New("usage : hexa new <destination> --module <chemin/de/module>")
	}

	plan, err := generator.PlanProject(positional[0], *module, *from)
	if err != nil {
		return fmt.Errorf("hexa new: %w", err)
	}
	report, err := generator.CreateProject(context.Background(), plan)
	if err != nil {
		return fmt.Errorf("hexa new: %w", err)
	}
	announceProject(plan, report)
	return nil
}

// commandFeature crée un module métier dans un projet existant.
func commandFeature(args []string) error {
	set := flag.NewFlagSet("make:feature", flag.ContinueOnError)
	set.SetOutput(os.Stderr)
	in := set.String("dans", ".", "racine du projet où créer le module")

	options, positional := generator.SplitArguments(args, generator.FlagsWithValue(set))
	if err := set.Parse(options); err != nil {
		return fmt.Errorf("arguments: %w", err)
	}
	if len(positional) != 1 {
		return errors.New("usage : hexa make:feature <nom_du_module>")
	}

	plan, err := generator.PlanFeature(positional[0], *in)
	if err != nil {
		return fmt.Errorf("hexa make:feature: %w", err)
	}
	if created := generator.CreateFeature(context.Background(), plan); created != nil {
		return fmt.Errorf("hexa make:feature: %w", created)
	}
	announceFeature(plan)
	return nil
}

// announceProject imprime ce qui a été créé et la suite.
func announceProject(plan generator.ProjectPlan, report generator.ProjectReport) {
	fmt.Printf("Projet créé dans %s\n", plan.Destination)
	fmt.Printf("  module   %s\n", plan.TargetModule)
	fmt.Printf("  fichiers %d\n", report.Files)
	if !report.GitInitialised {
		fmt.Fprint(os.Stderr, "hexa: dépôt git non initialisé — lancer `git init` et "+
			"`git config core.hooksPath .githooks` à la main\n")
	}
	fmt.Print(`
Ensuite :
  cd <destination>
  task init          # .env et outillage
  task check         # la barrière qualité, identique à la CI

Le module de démonstration ` + "`internal/modules/user_registration`" + ` est la
TRANCHE DE RÉFÉRENCE : sa forme est celle à copier pour écrire un module métier.
Le supprimer exige de retirer son montage de cmd/server.
`)
}

// announceFeature imprime les lignes à ajouter, et dit pourquoi elles ne sont
// pas ajoutées automatiquement.
func announceFeature(plan generator.FeaturePlan) {
	fmt.Printf("Module %s créé dans %s\n\n", plan.Dir, plan.Destination)
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
`, plan.Package, plan.Dir, plan.Package, plan.Package, plan.Package, plan.Package)
}
