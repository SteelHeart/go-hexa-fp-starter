package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnUnknownDriverOptionIsRefused : une clé d'option que le pilote ne lit pas
// refuse le démarrage.
//
// # Le défaut que ce test aurait attrapé
//
// Le deny-par-défaut s'arrêtait au nom du pilote. Mesuré avant correction (#93) :
// avec `bath_size` au lieu de `batch_size`, le serveur DÉMARRAIT, montait le
// module et n'en disait rien. Le dépileur tournait avec un lot par défaut que
// personne n'avait demandé.
//
// La cause est que les accesseurs d'options rendent la valeur par défaut quand
// la clé est absente — correct pris isolément — et que rien n'énumérait les clés
// connues. « Absente » et « mal orthographiée » étaient donc indiscernables :
// exactement la forme de défaut que ce dépôt rencontre à répétition.
func TestAnUnknownDriverOptionIsRefused(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: memory
    options:
      bath_size: 50
`)

	_, err := config.Load(catalogueAvecOptions())
	if err == nil {
		t.Fatal("une option inconnue doit refuser le démarrage")
	}

	message := err.Error()
	if !strings.Contains(message, "bath_size") {
		t.Errorf("le message doit nommer la clé fautive: %v", err)
	}
	// Refuser sans dire ce qui est admis transforme une garde en impasse : la
	// personne devant l'erreur doit deviner l'orthographe exacte.
	if !strings.Contains(message, "batch_size") {
		t.Errorf("le message doit lister les clés admises: %v", err)
	}
}

// catalogueAvecOptions déclare un module dont les pilotes admettent des options
// DIFFÉRENTES.
//
// La différence est le sujet : c'est elle qui permet de dire qu'une option
// valide pour un pilote ne l'est pas pour un autre.
func catalogueAvecOptions() config.ModuleCatalog {
	return config.ModuleCatalog{
		"facturation": {
			Default: "memory",
			Drivers: map[string]config.Resources{
				"memory": {Options: []string{"batch_size", "interval"}},
				"sqlite": {SQL: true, Options: []string{"batch_size", "path"}},
				"nu":     {},
			},
		},
	}
}
