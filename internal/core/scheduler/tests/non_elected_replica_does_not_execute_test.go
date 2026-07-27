package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestNonElectedReplicaDoesNotExecute est le test le plus important du module.
//
// Toute la valeur de l'ordonnanceur est là : N répliques, une seule exécution. Sans
// ce refus, un rappel part N fois et une facture est émise N fois — des défauts que
// le client découvre avant nous.
//
// Et « non élue » n'est PAS une erreur : c'est le cas nominal sur N-1 répliques, à
// chaque tick. Le compte rendu doit le dire ainsi, sinon les journaux ne
// contiendraient plus que ça.
func TestNonElectedReplicaDoesNotExecute(t *testing.T) {
	t.Parallel()

	var executed bool
	acquire := func(context.Context, domain.TaskName) (bool, error) { return false, nil }
	release := func(context.Context, domain.TaskName) error {
		t.Error("Release ne doit pas être appelée quand l'élection a échoué")
		return nil
	}

	log := &journal{}
	runner := newRunner(t, acquire, release, log)
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("rappel-quotidien"),
		Job:  func(context.Context) error { executed = true; return nil },
	})

	if executed {
		t.Error("une réplique non élue ne doit PAS exécuter le travail")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("aucun compte rendu")
	}
	if outcome.Event != domain.EventSkipped {
		t.Errorf("événement = %q, attendu %q", outcome.Event, domain.EventSkipped)
	}
	if outcome.Err != nil {
		t.Errorf("« non élue » n'est pas une erreur, reçu %v", outcome.Err)
	}
}
