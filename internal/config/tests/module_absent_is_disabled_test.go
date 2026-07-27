package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestModuleAbsentIsDisabled : deny par défaut jusque dans la configuration. On
// n'active jamais une capacité que personne n'a demandée.
func TestModuleAbsentIsDisabled(t *testing.T) {
	t.Parallel()

	// Resolve est l'étape qui pose les défauts, et elle est EXPLICITE depuis
	// l'ADR 014 : le catalogue vient du composition root, pas d'une table cachée.
	got := config.Modules{}.Resolve(shippedCatalog(t)).Get("outbox")
	if got.Enabled {
		t.Error("un module absent de la configuration doit être désactivé")
	}
	if got.Driver != "memory" {
		t.Errorf("pilote par défaut = %q, attendu \"memory\" (sans dépendance externe)", got.Driver)
	}
}
