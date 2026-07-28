package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestMissingDependencyRefusesStartup fait échouer le montage, pas la production.
//
// Sans ce refus, une dépendance oubliée produirait une déréférence de pointeur nil
// à la PREMIÈRE connexion réelle — donc en production, et avec une trace qui ne
// dit pas laquelle manquait. Le module refuse au démarrage et NOMME le port
// absent : c'est la différence entre trente secondes et une soirée.
func TestMissingDependencyRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	complete := deps(c)

	cases := map[string]auth.Deps{
		"HashSecret":   {VerifySecret: complete.VerifySecret, Now: complete.Now},
		"VerifySecret": {HashSecret: complete.HashSecret, Now: complete.Now},
		"Now":          {HashSecret: complete.HashSecret, VerifySecret: complete.VerifySecret},
		"tout":         {},
	}

	for name, incomplete := range cases {
		_, err := auth.New(config.Module{Enabled: true, Driver: "memory"}, incomplete)
		if !errors.Is(err, auth.ErrMissingDependency) {
			t.Errorf("sans %s : attendu ErrMissingDependency, obtenu %v", name, err)
		}
	}
}

// TestDisabledModuleNeedsNoDependency : un module éteint se monte sans rien.
//
// Exiger ses dépendances ferait échouer le démarrage du serveur à cause d'un
// module que personne n'a activé — et pousserait à câbler des dépendances
// factices « pour que ça passe », ce qui est exactement la manière dont un module
// désactivé finit par être activé sans qu'on s'en aperçoive.
func TestDisabledModuleNeedsNoDependency(t *testing.T) {
	t.Parallel()

	if _, err := auth.New(config.Module{Enabled: false}, auth.Deps{}); err != nil {
		t.Fatalf("un module désactivé ne doit exiger aucune dépendance : %v", err)
	}
}
