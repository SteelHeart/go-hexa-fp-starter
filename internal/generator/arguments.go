package generator

import (
	"flag"
	"strings"
)

// FlagsWithValue rend les options d'un jeu, sous leurs deux écritures.
//
// Dérivé du FlagSet plutôt qu'écrit à la main, et c'est le point : une table
// écrite à la main énumérait `-module` et `-depuis`. L'ajout de `--dans` par
// `make:feature` n'y figurait pas, donc sa valeur était classée comme
// positionnelle et la commande refusait une option pourtant écrite. Une seconde
// liste des mêmes options ne peut que diverger de la première.
func FlagsWithValue(set *flag.FlagSet) map[string]bool {
	withValue := map[string]bool{}
	set.VisitAll(func(f *flag.Flag) {
		withValue["-"+f.Name] = true
		withValue["--"+f.Name] = true
	})
	return withValue
}

// SplitArguments range les options d'un côté, les positionnels de l'autre.
//
// Le paquet `flag` s'arrête au PREMIER argument non-option : sans ce tri,
// `hexa new ./projet --module x` ignorerait `--module` en silence, et la
// commande échouerait en accusant l'absence d'une option pourtant écrite.
//
// Imposer l'ordre `--module x ./projet` serait une friction gratuite dans un
// outil dont la raison d'être est d'en supprimer.
func SplitArguments(args []string, withValue map[string]bool) (options, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case !strings.HasPrefix(arg, "-"):
			positional = append(positional, arg)
		case withValue[arg] && i+1 < len(args):
			// Forme `--module x` : la valeur suit, elle n'est pas positionnelle.
			options = append(options, arg, args[i+1])
			i++
		default:
			// Forme `--module=x`, ou option inconnue que `flag` refusera lui-même.
			options = append(options, arg)
		}
	}
	return options, positional
}
