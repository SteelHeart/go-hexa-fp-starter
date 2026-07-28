package main

import (
	"slices"
	"testing"
)

// TestTouteOptionDeclareeEstReconnue : le tri des arguments connaît les options
// de CHAQUE commande, pas seulement celles de `new`.
//
// # Le défaut que ce test aurait attrapé
//
// `separerArguments` portait une table écrite à la main : `-module`, `--module`,
// `-depuis`, `--depuis`. Quand `make:feature` a introduit `--dans`, la table ne
// le connaissait pas — sa valeur était donc classée comme POSITIONNELLE, et la
// commande refusait avec :
//
//	flag needs un argument: -dans
//
// Un message qui accuse l'utilisateur d'avoir omis ce qu'il vient d'écrire.
//
// La correction n'est pas d'ajouter `--dans` à la table : c'est de supprimer la
// table. `flagsAValeur` dérive les options du FlagSet, donc une option déclarée
// est reconnue par construction. Ce test le vérifie sur une option qui
// n'existait pas quand le tri a été écrit.
func TestTouteOptionDeclareeEstReconnue(t *testing.T) {
	t.Parallel()

	aValeur := optionsDe(t, "make:feature", "dans")

	options, positionnels := separerArguments([]string{"order_tracking", "--dans", "/projet"}, aValeur)

	if !slices.Equal(options, []string{"--dans", "/projet"}) {
		t.Errorf("options = %v, attendu la paire --dans/valeur", options)
	}
	if !slices.Equal(positionnels, []string{"order_tracking"}) {
		t.Errorf("positionnels = %v, attendu le seul nom de module", positionnels)
	}
}
