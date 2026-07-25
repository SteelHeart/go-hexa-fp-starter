package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestCancellationIsACleanStop : l'annulation est la fin NORMALE d'un ordonnanceur.
//
// Un arrêt propre ne doit ressembler à une panne ni dans le code de retour, ni dans
// les comptes rendus. Sinon chaque redéploiement remplit les alertes, et l'équipe
// apprend à ignorer les erreurs de l'ordonnanceur — y compris les vraies.
//
// Le test ne dort pas : il se termine dès la première exécution, déclenchée par une
// période très courte, puis annule.
func TestCancellationIsACleanStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	executed := make(chan struct{}, 1)
	acquire, release := alwaysElected()
	log := &journal{}
	runner := newRunner(t, acquire, release, log)

	quick := domain.Task{Name: "rapide", Every: time.Millisecond, Timeout: time.Second}
	stopped := make(chan error, 1)
	go func() {
		stopped <- runner.Run(ctx, []application.Scheduled{{
			Task: quick,
			Job: func(context.Context) error {
				select {
				case executed <- struct{}{}:
				default:
				}
				return nil
			},
		}})
	}()

	<-executed // la boucle tourne réellement
	cancel()

	if err := <-stopped; err != nil {
		t.Errorf("Run = %v, un arrêt demandé doit rendre nil", err)
	}
	if log.count(domain.EventSucceeded) == 0 {
		t.Error("aucune exécution réussie n'a été rapportée")
	}
	if log.count(domain.EventFailed) != 0 {
		t.Errorf("un arrêt propre ne doit produire aucun échec, reçu %d", log.count(domain.EventFailed))
	}
}
