package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// TestDispatcherStopsCleanlyOnCancellation : l'annulation est la fin NORMALE d'un
// worker.
//
// Un arrêt propre ne doit ressembler à une panne ni dans le code de retour, ni dans
// les alertes. Sinon chaque redéploiement en produit une, et l'équipe apprend à
// ignorer les erreurs du dépileur — y compris les vraies, celles qui signalent un
// événement définitivement perdu.
//
// Le test n'attend pas : il annule dès la première publication observée.
func TestDispatcherStopsCleanlyOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	published := make(chan struct{}, 1)
	claim := func(context.Context, int) ([]domain.Message, error) {
		return []domain.Message{pending("m-1", 0)}, nil
	}
	var handle ports.Handler = func(context.Context, domain.Message) error {
		select {
		case published <- struct{}{}:
		default:
		}
		return nil
	}

	observed := &spy{}
	dispatcher := newDispatcher(t, dispatcherPorts(observed, claim, handle), testPolicy())

	stopped := make(chan error, 1)
	go func() { stopped <- dispatcher.Run(ctx) }()

	<-published // la boucle tourne réellement
	cancel()

	if err := <-stopped; err != nil {
		t.Errorf("Run = %v, un arrêt demandé doit rendre nil", err)
	}
}
