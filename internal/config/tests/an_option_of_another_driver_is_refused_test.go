package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnOptionOfAnotherDriverIsRefused : une option correctement orthographiée,
// mais qui appartient à un AUTRE pilote, est refusée elle aussi.
//
// C'est le cas le plus coûteux, et le moins évident. La clé existe, elle est
// documentée, elle se relit sans soupçon — mais le pilote retenu ne la lit
// jamais. Le réglage n'a aucun effet, et rien ne le dit.
//
// Le cas réel est dans `dynconf` : `flags` et `settings` portent les valeurs
// VERSIONNÉES, donc n'ont de sens que pour le pilote fichier. Sous le pilote
// postgres, les valeurs vivent en base ; les écrire dans le dépôt décrirait une
// intention que rien n'exécute.
//
// C'est ce que la déclaration PAR PILOTE permet d'exprimer, là où une liste par
// module ne le pourrait pas.
func TestAnOptionOfAnotherDriverIsRefused(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: memory
    options:
      path: /tmp/facturation.db
`)

	_, err := config.Load(catalogueAvecOptions())
	if err == nil {
		t.Fatal("une option d'un autre pilote doit refuser le démarrage")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Errorf("le message doit nommer la clé fautive: %v", err)
	}
	if !strings.Contains(err.Error(), `"memory"`) {
		t.Errorf("le message doit nommer le pilote qui ne la lit pas: %v", err)
	}
}

// TestADriverWithoutOptionsRefusesEveryOne : un pilote qui ne déclare aucune
// option n'en accepte aucune, et le dit dans ces termes.
//
// La valeur zéro de `Resources` n'admet rien, et c'est le bon défaut : un pilote
// qui oublierait de déclarer ses options verrait sa configuration refusée — ce
// qui se remarque. L'inverse rouvrirait le trou pour tous les pilotes à la fois,
// silencieusement.
func TestADriverWithoutOptionsRefusesEveryOne(t *testing.T) {
	withCatalogTestConfig(t, `modules:
  facturation:
    enabled: true
    driver: nu
    options:
      batch_size: 50
`)

	_, err := config.Load(catalogueAvecOptions())
	if err == nil {
		t.Fatal("un pilote sans option déclarée doit tout refuser")
	}
	// « admises: » suivi du vide se lirait comme un message tronqué, et enverrait
	// chercher un défaut de l'outil plutôt que la faute de configuration.
	if !strings.Contains(err.Error(), "n'accepte aucune option") {
		t.Errorf("le message doit dire que ce pilote n'a aucune option: %v", err)
	}
}
