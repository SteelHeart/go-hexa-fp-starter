// Command hexa est l'outil en ligne de commande du socle.
//
// # Ce qu'il est aujourd'hui, et ce qu'il sera
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
//
// Le jour où le noyau s'importe, cette commande cessera de recopier `core/`
// pour l'ajouter en dépendance. C'est un travail assumé, et plus petit que
// celui d'une frontière mal placée.
package main

import (
	"fmt"
	"os"
)

// version est injectée à la compilation, comme pour les deux autres binaires.
var version = "dev"

const usage = `hexa — outil du socle hexagonal

  hexa new <destination> --module <chemin/de/module>   crée un projet
  hexa version                                          affiche la version

Exemple :
  hexa new ./mon-projet --module github.com/impactone/facturation

Options de « new » :
  --module   OBLIGATOIRE. Le chemin de module Go du projet créé.
  --depuis   Racine du socle à recopier (défaut : le répertoire courant).
             Ce doit être un dépôt git : ce sont les fichiers SUIVIS qui
             définissent le socle, ce qui écarte .git/, bin/ et .env.
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
		return commandeNew(args[1:])
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
