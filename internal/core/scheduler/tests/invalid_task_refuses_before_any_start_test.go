package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestInvalidTaskRefusesBeforeAnyStart : tout est validé AVANT que quoi que ce soit
// démarre.
//
// Lancer trois tâches sur quatre puis échouer laisserait un ordonnanceur à moitié
// vivant : certaines tâches tournent, l'appelant a reçu une erreur, et personne ne
// sait dans quel état on est. C'est l'état le plus difficile à diagnostiquer, donc
// celui qu'on refuse de produire.
//
// Une période nulle mérite une mention à part : `time.NewTicker(0)` PANIQUE, et une
// panique dans une goroutine d'ordonnanceur emporte tout le processus.
func TestInvalidTaskRefusesBeforeAnyStart(t *testing.T) {
	t.Parallel()

	valid := application.Scheduled{Task: task("bonne"), Job: func(context.Context) error { return nil }}
	noop := func(context.Context) error { return nil }

	cases := map[string][]application.Scheduled{
		"période nulle": {{
			Task: domain.Task{Name: "sans-periode", Every: 0}, Job: noop,
		}},
		"période négative": {{
			Task: domain.Task{Name: "negative", Every: -time.Second}, Job: noop,
		}},
		"nom manquant": {{
			Task: domain.Task{Every: time.Hour}, Job: noop,
		}},
		"délai négatif": {{
			Task: domain.Task{Name: "delai", Every: time.Hour, Timeout: -time.Second}, Job: noop,
		}},
		"la mauvaise est la seconde": {valid, {
			Task: domain.Task{Name: "sans-periode", Every: 0}, Job: noop,
		}},
	}

	for name, scheduled := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			acquire, release := alwaysElected()
			runner := newRunner(t, acquire, release, &journal{})

			err := runner.Run(context.Background(), scheduled)
			if !errors.Is(err, domain.ErrInvalidTask) {
				t.Errorf("Run = %v, attendu ErrInvalidTask", err)
			}
		})
	}
}
