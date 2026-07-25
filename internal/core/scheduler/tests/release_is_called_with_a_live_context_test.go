package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestReleaseIsCalledWithALiveContext : la libération ne doit pas hériter d'un
// contexte déjà mort.
//
// Le piège est réel et fréquent : le travail épuise son délai, le contexte
// d'exécution est annulé, et la libération faite avec ce même contexte échoue
// instantanément. Le verrou reste alors pris jusqu'à sa péremption — c'est-à-dire
// jusqu'à la mort de la réplique pour un verrou de session — et la tâche ne tourne
// plus du tout. Un délai dépassé deviendrait une panne définitive.
func TestReleaseIsCalledWithALiveContext(t *testing.T) {
	t.Parallel()

	var releaseErr error
	acquire := func(context.Context, domain.TaskName) (bool, error) { return true, nil }
	release := func(ctx context.Context, _ domain.TaskName) error {
		releaseErr = ctx.Err()
		return nil
	}

	log := &journal{}
	runner := newRunner(t, acquire, release, log)

	// Une tâche dont le délai est minuscule et dont le travail attend l'annulation :
	// le contexte d'exécution est donc mort au moment de la libération.
	expired := domain.Task{Name: "lente", Every: time.Hour, Timeout: time.Millisecond}
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: expired,
		Job: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})

	if releaseErr != nil {
		t.Errorf("Release a reçu un contexte annulé (%v) : le verrou resterait pris", releaseErr)
	}
}
