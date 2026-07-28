package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/exit"
)

// Profils de peuplement connus.
//
// Une table close plutôt qu'une chaîne libre : un profil mal orthographié doit
// REFUSER, jamais se rabattre sur le plus petit. Se rabattre créerait deux
// comptes là où l'on en attendait mille, et le défaut ne se verrait qu'au moment
// où un test de charge rendrait des chiffres flatteurs.
func profils() map[string]int {
	return map[string]int{
		// dev : de quoi cliquer dans une interface sans attendre.
		"dev": 3,
		// demo : de quoi montrer une liste paginée qui pagine vraiment.
		"demo": 25,
	}
}

// secretDeSeed est le mot de passe des comptes engendrés.
//
// # Pourquoi une constante ici est acceptable, et seulement ici
//
// Parce que `seed` REFUSE de s'exécuter hors développement et test — le garde
// est dans le composition root, avec son test. Ces comptes n'existent donc que
// sur des machines où ils ne protègent rien.
//
// Ce serait inacceptable partout ailleurs : un secret par défaut dans un
// artefact versionné est un secret déployé. C'est la raison pour laquelle
// l'amorçage de `auth` ENGENDRE le sien (ADR 017 §6) — là, le compte survit à
// l'environnement où il est né.
//
//nolint:gosec // mot de passe de peuplement : `seed` REFUSE hors développement et test
const secretDeSeed = "graine-de-developpement"

// Seed peuple le socle par les CAS D'USAGE, jamais par un `INSERT`.
//
// # Pourquoi le passage par les cas d'usage n'est pas négociable
//
// Un jeu de données fabriqué en SQL contourne les règles du domaine. Il produit
// des comptes que le code n'aurait jamais laissé naître : mots de passe non
// hachés, adresses non normalisées, statuts impossibles. Le jour où un test
// s'appuie dessus, il valide un état que la production ne peut pas atteindre —
// et il restera vert en cachant une vraie régression.
//
// Le corollaire coûte : peupler par les cas d'usage est LENT, parce que chaque
// compte paie un hachage Argon2id. C'est assumé — la lenteur est le prix de la
// représentativité, et un profil `perf` qui exigerait mille comptes demanderait
// un chemin distinct, pas un contournement de celui-ci.
func (c Command) Seed(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(c.Err)
	profil := fs.String("profile", "dev", "profil de peuplement: dev, demo")
	fs.Usage = func() {
		fmt.Fprintln(c.Err, "usage: hexa-cli seed --profile <dev|demo>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return exit.Usage
	}

	combien, connu := profils()[*profil]
	if !connu {
		fmt.Fprintf(c.Err, "erreur: profil inconnu %q — attendu dev ou demo\n", *profil)
		return exit.Usage
	}

	return c.peupler(ctx, *profil, combien)
}

// peupler crée les comptes du profil et rend le premier code d'échec.
//
// # S'arrêter au premier échec, plutôt que continuer
//
// Un peuplement à moitié fait est pire qu'un peuplement refusé : on ne sait pas
// ce qui existe. S'arrêter net laisse un état incomplet mais CONNU — le compte
// rendu dit combien ont été créés — là où continuer produirait des trous
// silencieux au milieu du jeu de données.
func (c Command) peupler(ctx context.Context, profil string, combien int) int {
	for i := 1; i <= combien; i++ {
		email := fmt.Sprintf("%s-%02d@example.test", profil, i)
		if code := c.inscrire(ctx, email, secretDeSeed); code != exit.OK {
			fmt.Fprintf(c.Err, "peuplement interrompu après %d compte(s) sur %d\n", i-1, combien)
			return code
		}
	}
	fmt.Fprintf(c.Err, "profil %s : %d compte(s) créé(s)\n", profil, combien)
	return exit.OK
}
