// Package outbox est le module noyau de publication garantie.
//
// C'est le composition root du module : le SEUL endroit qui connaît les pilotes.
// Un appelant reçoit des types fonction et ignore lequel est branché
// ([ADR 012](../../../documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "outbox"

// Pilotes de l'outbox.
//
// Nommés une fois pour que le montage et la table de partage
// (SharedAcrossProcesses) ne puissent pas parler de pilotes différents.
const (
	driverMemory   = "memory"
	driverPostgres = "postgres"
)

// Module expose les ports de l'outbox.
type Module struct {
	Enqueue      ports.Enqueue
	Claim        ports.Claim
	MarkDone     ports.MarkDone
	MarkFailed   ports.MarkFailed
	PendingCount ports.PendingCount
}

// Deps porte les dépendances que les pilotes peuvent réclamer.
//
// Pool peut être nil : le pilote `memory` n'en a pas besoin, et c'est
// précisément ce qui permet de démarrer sans base.
type Deps struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

// ErrDisabled signale un appel à un module désactivé.
//
// Un module désactivé échoue explicitement plutôt que de se rabattre sur un
// comportement inerte : un événement silencieusement ignoré est le pire défaut
// possible, parce qu'il ne se signale jamais.
var ErrDisabled = errors.New("module outbox désactivé dans config/modules.yaml")

// ErrPoolRequired signale un pilote qui exige une base absente.
var ErrPoolRequired = errors.New("le pilote postgres exige une connexion à la base")

// New construit le module selon la configuration.
//
// Un pilote inconnu refuse le démarrage : la validation de configuration l'a
// déjà rejeté, et ce second refus garantit qu'aucun chemin ne contourne le
// premier.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverMemory:
		store := memory.New(deps.Now)
		return Module{
			Enqueue:      store.Enqueue,
			Claim:        store.Claim,
			MarkDone:     store.MarkDone,
			MarkFailed:   store.MarkFailed,
			PendingCount: store.PendingCount,
		}, nil

	case driverPostgres:
		if deps.Pool == nil {
			return Module{}, ErrPoolRequired
		}
		store := postgres.New(deps.Pool)
		return Module{
			Enqueue:      store.Enqueue,
			Claim:        store.Claim,
			MarkDone:     store.MarkDone,
			MarkFailed:   store.MarkFailed,
			PendingCount: store.PendingCount,
		}, nil

	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

var errUnknownDriver = errors.New("pilote outbox inconnu")

// disabled retourne des ports qui refusent explicitement.
func disabled() Module {
	return Module{
		Enqueue: func(context.Context, domain.NewMessage) (domain.MessageID, error) {
			return "", ErrDisabled
		},
		Claim:        func(context.Context, int) ([]domain.Message, error) { return nil, ErrDisabled },
		MarkDone:     func(context.Context, domain.MessageID) error { return ErrDisabled },
		MarkFailed:   func(context.Context, domain.FailedAttempt) error { return ErrDisabled },
		PendingCount: func(context.Context) (int64, error) { return 0, ErrDisabled },
	}
}
