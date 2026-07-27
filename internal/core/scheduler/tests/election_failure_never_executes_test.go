package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestElectionFailureNeverExecutes : dans le doute, on n'exécute pas.
//
// Une base injoignable ne dit pas « personne ne tient la tâche », elle ne dit rien.
// Exécuter quand même reviendrait à supposer qu'on est seul — supposition fausse
// dès qu'il y a deux répliques, et qui produit exactement la double exécution que
// le module existe pour empêcher.
//
// Le repli va donc dans le sens sûr : ne pas exécuter est bénin, exécuter deux fois
// ne l'est pas.
func TestElectionFailureNeverExecutes(t *testing.T) {
	t.Parallel()

	panne := errors.New("base injoignable")
	acquire := func(context.Context, domain.TaskName) (bool, error) { return false, panne }
	release := func(context.Context, domain.TaskName) error {
		t.Error("Release ne doit pas être appelée quand l'élection est en panne")
		return nil
	}

	var executed bool
	log := &journal{}
	runner := newRunner(t, acquire, release, log)
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("facturation"),
		Job:  func(context.Context) error { executed = true; return nil },
	})

	if executed {
		t.Error("une élection en panne ne doit PAS mener à une exécution")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("une panne d'élection doit être signalée, jamais avalée")
	}
	if outcome.Event != domain.EventElectionFailed {
		t.Errorf("événement = %q, attendu %q", outcome.Event, domain.EventElectionFailed)
	}
	if !errors.Is(outcome.Err, panne) {
		t.Errorf("cause = %v, attendu la panne d'origine", outcome.Err)
	}
}
