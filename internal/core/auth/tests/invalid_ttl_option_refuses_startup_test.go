package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestInvalidTTLOptionRefusesStartup : une durée illisible arrête le démarrage.
//
// Le repli sur la valeur par défaut serait pire que le refus. Quelqu'un écrit
// `session_ttl: 30` en pensant « trente minutes », le module retient douze heures,
// et la configuration affirme le contraire de ce qui s'applique. Personne ne
// relit une valeur qui n'a produit aucune erreur.
func TestInvalidTTLOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	for _, value := range []any{"trente minutes", "30x", true, []string{"1h"}} {
		_, err := auth.New(config.Module{
			Enabled: true,
			Driver:  "memory",
			Options: map[string]any{"session_ttl": value},
		}, deps(c))
		if err == nil {
			t.Errorf("session_ttl=%v (%T) : le démarrage doit être refusé", value, value)
		}
	}
}

// TestUnknownOptionRefusesStartup relie ce module à la garde de l'issue #93.
//
// Une option mal orthographiée était ignorée EN SILENCE : le serveur démarrait,
// montait le pilote, et n'en disait rien. Le catalogue de ce module énumère ses
// options ; le test constate que la garde couvre bien `auth`, plutôt que de faire
// confiance au fait qu'elle est branchée ailleurs.
func TestUnknownOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	allowed := auth.Catalog()[auth.Name].Drivers["memory"].Options
	if len(allowed) == 0 {
		t.Fatal("le catalogue doit énumérer les options du pilote memory")
	}

	found := false
	for _, option := range allowed {
		if option == auth.OptionSessionTTL {
			found = true
		}
	}
	if !found {
		t.Fatalf("le catalogue doit déclarer %q ; il déclare %v", auth.OptionSessionTTL, allowed)
	}
}
