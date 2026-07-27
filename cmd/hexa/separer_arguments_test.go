package main

import (
	"slices"
	"testing"
)

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

	for nom, tc := range cas {
		t.Run(nom, func(t *testing.T) {
			t.Parallel()
			options, positionnels := separerArguments(tc.args)
			if !slices.Equal(options, tc.voulOptions) {
				t.Errorf("options = %v, attendu %v", options, tc.voulOptions)
			}
			if !slices.Equal(positionnels, tc.voulPositionnel) {
				t.Errorf("positionnels = %v, attendu %v", positionnels, tc.voulPositionnel)
			}
		})
	}
}
