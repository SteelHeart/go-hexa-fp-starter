// Package scheduler est le module noyau des tâches périodiques.
//
// Composition root du module : le seul endroit qui connaît les pilotes.
//
// # Pourquoi l'élection est un port et la boucle une orchestration
//
// La version précédente de cette brique fondait le minuteur, l'élection par verrou
// consultatif et la journalisation dans un seul type. Conséquence : elle exigeait
// PostgreSQL pour répéter une tâche, y compris dans un binaire mono-instance qui
// n'a personne avec qui s'accorder. C'est exactement ce que l'ADR 012 interdit.
//
// Séparées, la boucle se teste sans base et le pilote par défaut n'a plus aucune
// dépendance.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/inproc"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "scheduler"

// Run lance l'ordonnanceur jusqu'à l'annulation du contexte.
//
// Déclaré ici et non dans `ports/` parce qu'il nomme `application.Scheduled` : les
// ports d'un module noyau ne dépendent que de leur domaine (.arch-go.yml §4e).
type Run = func(ctx context.Context, scheduled []application.Scheduled) error

// Module expose les ports de l'ordonnanceur.
type Module struct {
	// Run exécute les tâches. C'est l'entrée normale du module.
	Run Run
	// Acquire et Release sont exposés pour un déclencheur EXTERNE — tâche planifiée
	// d'un orchestrateur, minuteur du système — qui veut la même élection sans la
	// boucle.
	Acquire ports.Acquire
	Release ports.Release
}

// Deps porte les dépendances que les pilotes peuvent réclamer.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// ErrDisabled signale un appel à un module désactivé.
var ErrDisabled = errors.New("module scheduler désactivé dans config/modules.yaml")

// ErrPoolRequired signale un pilote qui exige une base absente.
var ErrPoolRequired = errors.New("le pilote advisory-lock exige une connexion à la base")

// ErrLoggerRequired signale l'absence de journal.
//
// L'orchestration ne journalise pas : elle rend compte (ports.Report). Le compte
// rendu doit bien aboutir quelque part, sinon une tâche en échec échouerait en
// silence — le pire défaut possible pour un travail qui tourne sans témoin.
var ErrLoggerRequired = errors.New("le module scheduler exige un journal pour rendre compte")

var errUnknownDriver = errors.New("pilote scheduler inconnu")

// New construit le module selon la configuration.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}

	elected, err := elector(cfg, deps)
	if err != nil {
		return Module{}, err
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}
	runner, err := application.NewRunner(application.Ports{
		Acquire: elected.acquire,
		Release: elected.release,
		Report:  LogReport(deps.Logger),
		Now:     now,
	})
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s: %w", Name, err)
	}
	return Module{Run: runner.Run, Acquire: elected.acquire, Release: elected.release}, nil
}

// election porte le couple indissociable acquisition / libération.
//
// Regroupé en type plutôt qu'en deux valeurs de retour : les deux fonctions n'ont
// aucun sens séparées — acquérir sans pouvoir libérer gèle une tâche pour toujours.
// Le regroupement rend aussi impossible d'inverser deux retours que le compilateur
// ne distinguerait pas.
type election struct {
	acquire ports.Acquire
	release ports.Release
}

// elector choisit le mécanisme d'élection.
func elector(cfg config.Module, deps Deps) (election, error) {
	switch cfg.Driver {
	case "cron-inproc":
		local := inproc.New()
		return election{acquire: local.Acquire, release: local.Release}, nil
	case "advisory-lock":
		if deps.Pool == nil {
			return election{}, ErrPoolRequired
		}
		shared := postgres.New(deps.Pool)
		return election{acquire: shared.Acquire, release: shared.Release}, nil
	default:
		return election{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// disabled retourne des ports qui refusent explicitement.
//
// Acquire rend `false` ET une erreur : `false` garantit qu'aucune tâche ne
// s'exécute, l'erreur garantit que le refus se voit.
func disabled() Module {
	return Module{
		Run:     func(context.Context, []application.Scheduled) error { return ErrDisabled },
		Acquire: func(context.Context, domain.TaskName) (bool, error) { return false, ErrDisabled },
		Release: func(context.Context, domain.TaskName) error { return ErrDisabled },
	}
}
