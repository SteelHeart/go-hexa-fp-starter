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
func New(cfg config.Module, deps Deps) (Module, error) {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if !cfg.Enabled {
		return Module{Record: func(context.Context, domain.Entry) error { return ErrDisabled }}, nil
	}

	switch cfg.Driver {
	case "log":
		if deps.Logger == nil {
			return Module{}, ErrLoggerRequired
		}
		return Module{Record: logdriver.New(deps.Logger, deps.Now)}, nil
	case "postgres":
		if deps.Pool == nil {
			return Module{}, ErrPoolRequired
		}
		return Module{Record: pgdriver.New(deps.Pool, deps.Now)}, nil
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}
