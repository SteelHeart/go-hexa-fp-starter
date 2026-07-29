// Commande cli expose le socle sur la surface LIGNE DE COMMANDE.
//
// # Pourquoi ce binaire existe
//
// Pour démontrer la propriété n°2 du socle — *le nombre de frontends est un
// non-sujet* — plutôt que de l'énoncer. Avec la seule surface HTTP, elle était
// écrite ; avec celle-ci, elle est **mesurée** : la carte d'impact de son ajout
// ne touche que `adapters/primary/`, et pas une ligne de `domain/`, `ports/` ou
// `application/`.
//
// La commande `register` appelle EXACTEMENT le même port que `POST /v1/users`.
// Le module ne sait pas laquelle des deux surfaces l'appelle.
//
// # C'est un composition root, comme `cmd/server` et `cmd/worker`
//
// Il a le droit de tout connaître (ADR 004), et il lit la MÊME configuration :
// un module activé ici l'est là-bas, avec le même pilote. Un binaire
// d'administration qui aurait sa propre configuration finirait par agir sur un
// autre magasin que celui qu'il prétend administrer.
//
// # Codes de sortie
//
// `sysexits.h`, parce qu'un binaire de ligne de commande est appelé par des
// scripts plus souvent que par des humains, et qu'un script lit le code de
// retour pour décider s'il faut RÉESSAYER. Voir internal/pkg/exit.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

func main() { os.Exit(demarrer()) }

// demarrer installe l'écoute des signaux et rend le code de sortie.
//
// Séparée de `main` parce qu'`os.Exit` NE DÉCLENCHE AUCUN `defer` : les poser
// dans `main` autour d'un `os.Exit` revient à ne pas les poser du tout. Ici, la
// fonction rend un entier, tous ses `defer` s'exécutent, et `main` ne fait plus
// que sortir.
func demarrer() int {
	// NotifyContext dès le départ : un Ctrl+C pendant un `seed` doit
	// interrompre proprement, pas laisser la moitié des comptes créés sans que
	// personne sache lesquels.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, os.Args[1:])
}

// run dispatche la sous-commande et rend son code de sortie.
//
// Rend un ENTIER plutôt que d'appeler `os.Exit` lui-même : c'est ce qui rend le
// dispatch vérifiable par un test, et ce qui garantit que les `defer` posés en
// amont s'exécutent — `os.Exit` les saute tous, y compris la fermeture d'un pool.
func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		usage()
		return exit.Usage
	}

	switch args[0] {
	case "register":
		return commandRegister(ctx, args[1:])
	case "seed":
		return commandSeed(ctx, args[1:])
	case "health":
		return commandHealth(ctx)
	case "-h", "--help", "help":
		usage()
		return exit.OK
	default:
		fmt.Fprintf(os.Stderr, "commande inconnue: %q\n\n", args[0])
		usage()
		return exit.Usage
	}
}

// usage décrit ce que ce binaire sait faire.
//
// `migrate:status` n'y figure PAS, et c'est une décision : l'état des migrations
// est déjà rendu par `goose`, que le `Taskfile` et la CI appellent. Le
// réimplémenter ici créerait une seconde source de vérité sur l'état du schéma —
// et le jour où les deux divergeraient, c'est celle qui ment qu'on croirait.
func usage() {
	fmt.Fprint(os.Stderr, `usage: hexa-cli <commande> [options]

  register --email <adresse>   inscrire un compte ; le mot de passe est lu sur stdin
  seed --profile <dev|demo>    peupler le socle PAR LES CAS D'USAGE — jamais en production
  health                       vérifier que ce binaire pourrait démarrer

L'état des migrations est rendu par `+"`task migrate:status`"+`, pas par cette commande :
une seconde source de vérité sur le schéma est une source de vérité de trop.
`)
}
