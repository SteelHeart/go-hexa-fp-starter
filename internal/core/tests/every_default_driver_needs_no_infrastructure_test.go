package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
)

// TestEveryDefaultDriverNeedsNoInfrastructure tient la promesse centrale de
// l'ADR 012 : `hexa new` puis `go run`, ça démarre.
//
// Tous les modules noyau activés d'un coup, tous sur le pilote par défaut de
// LEUR PROPRE catalogue : le résultat ne doit exiger ni base, ni cache.
//
// Le jour où quelqu'un fait de `postgres` le défaut d'un module — par
// commodité, parce que c'est le pilote le plus complet — ce test échoue. C'est
// le seul endroit qui s'en apercevra avant les utilisateurs.
//
// Il vivait dans `internal/config/internal_test.go`, où il interrogeait une
// table du framework. Depuis l'ADR 014 il n'y a plus de table : il interroge
// donc les catalogues RÉELS des six modules, ce qu'il aurait dû faire depuis le
// début.
func TestEveryDefaultDriverNeedsNoInfrastructure(t *testing.T) {
	t.Parallel()

	catalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("catalogue du noyau: %v", err)
	}

	all := config.Modules{}
	for name := range catalog {
		all[name] = config.Module{Enabled: true}
	}
	all = all.Resolve(catalog)

	if all.RequiresSQL(catalog) {
		t.Error("les pilotes par défaut ne doivent exiger aucune base")
	}
	if all.RequiresCache(catalog) {
		t.Error("les pilotes par défaut ne doivent exiger aucun cache")
	}
}
