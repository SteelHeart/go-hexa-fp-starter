package tests

import (
	"slices"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
)

// TestEveryModuleDefaultDriverIsDeclared : le pilote par défaut d'un module
// doit figurer parmi ses pilotes admis.
//
// Un défaut absent de la liste ferait refuser le démarrage d'un module pourtant
// correctement déclaré, avec un message qui accuserait l'utilisateur d'une
// faute qui est la nôtre.
//
// Un défaut VIDE produirait le même refus, en pire : le message citerait un
// pilote `""` que personne n'a écrit.
func TestEveryModuleDefaultDriverIsDeclared(t *testing.T) {
	t.Parallel()

	catalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("catalogue du noyau: %v", err)
	}
	if len(catalog) == 0 {
		t.Fatal("catalogue vide : ce test ne vérifierait rien")
	}

	for name := range catalog {
		allowed := catalog.AllowedDrivers(name)
		fallback := catalog.DefaultDriver(name)
		if fallback == "" {
			t.Errorf("module %q : aucun pilote par défaut", name)
			continue
		}
		if !slices.Contains(allowed, fallback) {
			t.Errorf("module %q : défaut %q absent des pilotes admis %v", name, fallback, allowed)
		}
	}
}
