package main

import (
	"flag"
	"io"
	"slices"
	"testing"
)

// optionsDe construit le jeu d'options à partir d'une commande RÉELLE.
//
// Le test ne redéclare donc pas la liste des options : il la dérive comme le
// fait le code. Une liste écrite ici finirait par diverger de celle des
// commandes — c'est exactement ce qui s'est produit avec la table figée que
// `flagsAValeur` remplace.
func optionsDe(t *testing.T, nom string, noms ...string) map[string]bool {
	t.Helper()
	jeu := flag.NewFlagSet(nom, flag.ContinueOnError)
	jeu.SetOutput(io.Discard)
	for _, n := range noms {
		jeu.String(n, "", "")
	}
	return flagsAValeur(jeu)
}

// TestSepererArguments : le paquet `flag` s'arrête au PREMIER argument
// non-option.
//
// Sans ce tri, `hexa new ./projet --module x` ignorerait `--module` en silence,
// et la commande échouerait en accusant l'absence d'une option pourtant écrite —
// le pire message possible, celui qui envoie corriger ce qui est déjà correct.
func TestSepererArguments(t *testing.T) {
	t.Parallel()

	cas := map[string]struct {
		args            []string
		voulOptions     []string
		voulPositionnel []string
	}{
		"destination avant l'option": {
			args:            []string{"./projet", "--module", "x/y"},
			voulOptions:     []string{"--module", "x/y"},
			voulPositionnel: []string{"./projet"},
		},
		"destination apres l'option": {
			args:            []string{"--module", "x/y", "./projet"},
			voulOptions:     []string{"--module", "x/y"},
			voulPositionnel: []string{"./projet"},
		},
		"forme collee": {
			args:            []string{"./projet", "--module=x/y"},
			voulOptions:     []string{"--module=x/y"},
			voulPositionnel: []string{"./projet"},
		},
		"un seul tiret": {
			args:            []string{"-module", "x/y", "./projet"},
			voulOptions:     []string{"-module", "x/y"},
			voulPositionnel: []string{"./projet"},
		},
		"deux options": {
			args:            []string{"./projet", "--module", "x/y", "--depuis", "/socle"},
			voulOptions:     []string{"--module", "x/y", "--depuis", "/socle"},
			voulPositionnel: []string{"./projet"},
		},
		"option inconnue laissee a flag": {
			args:            []string{"--inconnue", "./projet"},
			voulOptions:     []string{"--inconnue"},
			voulPositionnel: []string{"./projet"},
		},
	}

	aValeur := optionsDe(t, "new", "module", "depuis")

	for nom, tc := range cas {
		t.Run(nom, func(t *testing.T) {
			t.Parallel()
			options, positionnels := separerArguments(tc.args, aValeur)
			if !slices.Equal(options, tc.voulOptions) {
				t.Errorf("options = %v, attendu %v", options, tc.voulOptions)
			}
			if !slices.Equal(positionnels, tc.voulPositionnel) {
				t.Errorf("positionnels = %v, attendu %v", positionnels, tc.voulPositionnel)
			}
		})
	}
}
