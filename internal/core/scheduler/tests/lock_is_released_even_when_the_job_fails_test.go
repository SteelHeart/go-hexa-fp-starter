package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockIsReleasedEvenWhenTheJobFails : un travail en échec libère quand même
// l'élection.
//
// Sans cette libération, la première erreur d'une tâche la gèlerait jusqu'à la mort
// de la réplique. Le remède serait pire que le mal : une tâche qui échoue une fois
// doit réessayer au tick suivant, pas disparaître.
func TestLockIsReleasedEvenWhenTheJobFails(t *testing.T) {
	t.Parallel()

	var released bool
	acquire := func(context.Context, domain.TaskName) (bool, error) { return true, nil }
	release := func(context.Context, domain.TaskName) error { released = true; return nil }

	log := &journal{}
	runner := newRunner(t, acquire, release, log)
	panne := errors.New("travail en échec")
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("purge"),
		Job:  func(context.Context) error { return panne },
	})

	if !released {
		t.Error("l'élection doit être libérée même après un échec du travail")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("aucun compte rendu")
	}
	if outcome.Event != domain.EventFailed {
		t.Errorf("événement = %q, attendu %q", outcome.Event, domain.EventFailed)
	}
	if !errors.Is(outcome.Err, panne) {
		t.Errorf("cause = %v, attendu la panne d'origine", outcome.Err)
	}
}
