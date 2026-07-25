package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestAdvisoryLockDriverRefusesWithoutPool : le pilote qui exige une base refuse au
// démarrage.
//
// Ce test verrouille aussi la correction d'un défaut de conception de la version
// précédente : elle exigeait PostgreSQL pour SIMPLEMENT répéter une tâche, y compris
// dans un binaire mono-instance qui n'a personne avec qui s'accorder. Désormais
// l'élection est un pilote, et le pilote par défaut n'exige rien — la base n'est
// requise que si on la choisit.
func TestAdvisoryLockDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := scheduler.New(
		config.Module{Enabled: true, Driver: "advisory-lock"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if !errors.Is(err, scheduler.ErrPoolRequired) {
		t.Errorf("erreur = %v, attendu ErrPoolRequired", err)
	}
}
