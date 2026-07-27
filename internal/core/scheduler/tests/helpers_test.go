// Package tests contient les tests en BOÎTE NOIRE du module scheduler : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// discardLogger satisfait l'exigence de journal sans polluer la sortie des tests.
//
// Le module refuse un logger nil : un compte rendu qui n'aboutit nulle part
// laisserait une tâche échouer en silence.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newInprocModule construit le module sur son pilote par défaut.
func newInprocModule(t *testing.T) scheduler.Module {
	t.Helper()
	mod, err := scheduler.New(
		config.Module{Enabled: true, Driver: "cron-inproc"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if err != nil {
		t.Fatalf("construction du module: %v", err)
	}
	return mod
}

// startedAt est l'instant de référence. Aucun test ne lit l'horloge réelle.
func startedAt() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// clock est une horloge pilotée : chaque lecture avance d'un pas fixe, ce qui rend
// les durées mesurées déterministes sans jamais attendre.
type clock struct {
	mu   sync.Mutex
	at   time.Time
	step time.Duration
}

func newClock(step time.Duration) *clock {
	return &clock{at: startedAt(), step: step}
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.at
	c.at = c.at.Add(c.step)
	return current
}

// journal collecte les comptes rendus. C'est ce que le port Report rend possible :
// vérifier une politique d'exécution en lisant des valeurs, pas du JSON.
type journal struct {
	mu       sync.Mutex
	outcomes []domain.Outcome
}

func (j *journal) report() ports.Report {
	return func(_ context.Context, outcome domain.Outcome) {
		j.mu.Lock()
		defer j.mu.Unlock()
		j.outcomes = append(j.outcomes, outcome)
	}
}

func (j *journal) all() []domain.Outcome {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]domain.Outcome(nil), j.outcomes...)
}

// count compte les comptes rendus d'un événement donné.
func (j *journal) count(event domain.Event) int {
	total := 0
	for _, outcome := range j.all() {
		if outcome.Event == event {
			total++
		}
	}
	return total
}

// last rend le dernier compte rendu, ou false s'il n'y en a aucun.
func (j *journal) last() (domain.Outcome, bool) {
	all := j.all()
	if len(all) == 0 {
		return domain.Outcome{}, false
	}
	return all[len(all)-1], true
}

// alwaysElected accorde toujours l'exécution.
func alwaysElected() (acquire ports.Acquire, release ports.Release) {
	return func(context.Context, domain.TaskName) (bool, error) { return true, nil },
		func(context.Context, domain.TaskName) error { return nil }
}

// newRunner construit un exécutant sur des closures.
func newRunner(t *testing.T, acquire ports.Acquire, release ports.Release, j *journal) *application.Runner {
	t.Helper()
	runner, err := application.NewRunner(application.Ports{
		Acquire: acquire,
		Release: release,
		Report:  j.report(),
		Now:     newClock(250 * time.Millisecond).now,
	})
	if err != nil {
		t.Fatalf("construction de l'exécutant: %v", err)
	}
	return runner
}

// task décrit une tâche valide.
func task(name string) domain.Task {
	return domain.Task{Name: domain.TaskName(name), Every: time.Hour, Timeout: time.Minute}
}
