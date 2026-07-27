// Package audit est le module noyau de journalisation d'audit.
//
// Composition root du module : le seul endroit qui connaît les pilotes.
package audit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	logdriver "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/log"
	pgdriver "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "audit"

// Noms des pilotes de ce module.
//
// Elles existent pour que `Catalog` et le `switch` de `New` partagent le MÊME
// identifiant. C'est ce qui rend la divergence entre les deux IMPOSSIBLE, là où
// l'ADR 014 ne promettait que de la rendre improbable — le compilateur refuse
// une constante qui n'existe pas, un littéral mal orthographié passe.
//
// Le linter `goconst` a signalé la répétition dès que le catalogue est arrivé.
// Il avait raison, et pour une raison plus forte que la sienne.
const (
	driverLog      = "log"
	driverPostgres = "postgres"
)

// Module expose le port d'audit.
type Module struct{ Record ports.Record }

// Deps porte les dépendances des pilotes.
//
// Pool peut être nil : le pilote `log` n'en a pas besoin.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// Erreurs du module.
var (
	ErrDisabled       = errors.New("module audit désactivé dans config/modules.yaml")
	ErrPoolRequired   = errors.New("le pilote postgres exige une connexion à la base")
	ErrLoggerRequired = errors.New("le pilote log exige un journal")
	errUnknownDriver  = errors.New("pilote audit inconnu")
)

// New construit le module selon la configuration.
//
// Un pilote inconnu refuse le démarrage : la validation de configuration l'a déjà
// rejeté, et ce second refus garantit qu'aucun chemin ne contourne le premier.
func New(cfg config.Module, deps Deps) (Module, error) {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverLog:
		return newLog(deps)
	case driverPostgres:
		return newPostgres(deps)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// disabled rend un module qui refuse à l'appel.
//
// Un audit désactivé ne doit pas rendre `nil` en silence : une trace d'audit qu'on
// croit écrite et qui ne l'est pas est pire que pas d'audit du tout.
func disabled() Module {
	return Module{Record: func(context.Context, domain.Entry) error { return ErrDisabled }}
}

func newLog(deps Deps) (Module, error) {
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}
	return Module{Record: logdriver.New(deps.Logger, deps.Now)}, nil
}

func newPostgres(deps Deps) (Module, error) {
	if deps.Pool == nil {
		return Module{}, ErrPoolRequired
	}
	return Module{Record: pgdriver.New(deps.Pool, deps.Now)}, nil
}
