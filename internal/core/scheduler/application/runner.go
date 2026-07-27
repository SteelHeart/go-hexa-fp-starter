// Package application orchestre l'exécution des tâches périodiques.
//
// Ce paquet ne connaît AUCUN pilote et ne fait AUCUNE I/O : il reçoit l'élection,
// le compte rendu et l'horloge sous forme de types fonction. C'est ce qui permet de
// le tester avec des closures, sans base et sans attendre — la politique
// d'exécution est la partie qu'on veut vraiment prouver.
//
// Conformément à `rules/README.md` § « le cœur est pur » : ni `time.Now()`, ni
// logger, ni `panic` ici. `time.NewTicker` reste, parce qu'un minuteur n'est ni une
// lecture d'horloge ni une entrée-sortie — et parce qu'un ordonnanceur sans
// minuteur n'ordonnance rien.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// Scheduled associe une description de tâche au travail à exécuter.
type Scheduled struct {
	Task domain.Task
	Job  ports.Job
}

// Ports porte les contrats dont l'orchestration a besoin.
//
// Regroupés dans une structure plutôt qu'en paramètres : quatre arguments
// positionnels de même forme — trois fonctions et une horloge — s'inversent au
// premier ajout, et le compilateur ne dirait rien.
type Ports struct {
	Acquire ports.Acquire
	Release ports.Release
	Report  ports.Report
	Now     ports.Now
}

// Runner exécute les tâches enregistrées.
type Runner struct{ ports Ports }

// ErrMissingPort refuse une orchestration incomplète.
var ErrMissingPort = errors.New("port manquant pour l'ordonnanceur")

// ErrNoJob refuse une tâche sans travail associé.
var ErrNoJob = errors.New("tâche planifiée sans travail")

// NewRunner construit l'exécutant.
//
// Refuse un port manquant plutôt que de laisser un nil paniquer au premier tick :
// une panique dans une goroutine d'ordonnanceur emporte tout le processus.
func NewRunner(p Ports) (*Runner, error) {
	if p.Acquire == nil || p.Release == nil || p.Report == nil || p.Now == nil {
		return nil, ErrMissingPort
	}
	return &Runner{ports: p}, nil
}

// Run exécute toutes les tâches jusqu'à l'annulation du contexte.
//
// Valide TOUT avant de démarrer quoi que ce soit : lancer trois tâches sur quatre
// puis échouer laisserait un ordonnanceur à moitié vivant, l'état le plus difficile
// à diagnostiquer.
func (r *Runner) Run(ctx context.Context, scheduled []Scheduled) error {
	if err := validate(scheduled); err != nil {
		return err
	}

	done := make(chan struct{}, len(scheduled))
	for _, entry := range scheduled {
		go func(s Scheduled) {
			defer func() { done <- struct{}{} }()
			r.loop(ctx, s)
		}(entry)
	}
	for range scheduled {
		<-done
	}

	// L'annulation est la fin NORMALE d'un ordonnanceur : un arrêt propre ne doit
	// ressembler à une panne ni dans les comptes rendus, ni dans le code de retour.
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("ordonnanceur interrompu: %w", err)
	}
	return nil
}

// validate refuse un lot de tâches inexploitable.
func validate(scheduled []Scheduled) error {
	seen := make(map[domain.TaskName]struct{}, len(scheduled))
	for _, entry := range scheduled {
		if err := entry.Task.Validate(); err != nil {
			return fmt.Errorf("tâche planifiée refusée: %w", err)
		}
		if entry.Job == nil {
			return fmt.Errorf("%w: %s", ErrNoJob, entry.Task.Name)
		}
		// Deux tâches de même nom partagent la clé d'élection : elles s'excluraient
		// mutuellement et l'une des deux ne tournerait jamais, sans jamais se
		// signaler autrement que par « skipped ».
		if _, duplicate := seen[entry.Task.Name]; duplicate {
			return fmt.Errorf("%w: %s déclarée deux fois", domain.ErrInvalidTask, entry.Task.Name)
		}
		seen[entry.Task.Name] = struct{}{}
	}
	return nil
}

// loop répète une tâche jusqu'à l'annulation.
func (r *Runner) loop(ctx context.Context, s Scheduled) {
	ticker := time.NewTicker(s.Task.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.RunOnce(ctx, s)
		}
	}
}

// RunOnce tente une exécution unique : élection, travail, libération.
//
// Exportée pour qu'un déclencheur EXTERNE — ordonnanceur du système, tâche
// planifiée d'un orchestrateur — applique exactement la même politique, et pour
// qu'un test la vérifie sans dépendre d'un minuteur.
func (r *Runner) RunOnce(ctx context.Context, s Scheduled) {
	runCtx, cancel := context.WithTimeout(ctx, s.Task.Deadline())
	defer cancel()

	elected, err := r.ports.Acquire(runCtx, s.Task.Name)
	if err != nil {
		r.ports.Report(runCtx, domain.Outcome{
			Task: s.Task.Name, Event: domain.EventElectionFailed, Err: err,
		})
		return
	}
	if !elected {
		r.ports.Report(runCtx, domain.Outcome{Task: s.Task.Name, Event: domain.EventSkipped})
		return
	}

	// La libération utilise un contexte SANS annulation : si le travail a épuisé le
	// délai, libérer avec le même contexte échouerait et le verrou resterait pris
	// jusqu'à sa péremption — la tâche ne tournerait plus pendant tout ce temps.
	defer func() {
		if err := r.ports.Release(context.WithoutCancel(runCtx), s.Task.Name); err != nil {
			r.ports.Report(runCtx, domain.Outcome{
				Task: s.Task.Name, Event: domain.EventReleaseFailed, Err: err,
			})
		}
	}()

	r.execute(runCtx, s)
}

// execute lance le travail et rend compte de son sort.
func (r *Runner) execute(ctx context.Context, s Scheduled) {
	started := r.ports.Now()
	err := s.Job(ctx)
	elapsed := r.ports.Now().Sub(started)

	event := domain.EventSucceeded
	if err != nil {
		event = domain.EventFailed
	}
	r.ports.Report(ctx, domain.Outcome{
		Task: s.Task.Name, Event: event, Duration: elapsed, Err: err,
	})
}
