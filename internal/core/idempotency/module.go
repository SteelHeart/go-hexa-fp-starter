// Package idempotency est le module noyau qui rend une écriture rejouable.
//
// C'est le composition root du module : le SEUL endroit qui connaît les pilotes.
// Un appelant reçoit des types fonction et ignore lequel est branché
// ([ADR 012](../../../documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/redis"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "idempotency"

// defaultTTL est la fenêtre pendant laquelle un rejeu est reconnu.
//
// Vingt-quatre heures couvre le cas qui motive le module : un mobile hors réseau
// qui reprend le lendemain. Au-delà, un rejeu recrée la ressource — c'est un
// arbitrage, pas un oubli, et il se règle par `options.ttl`.
const defaultTTL = 24 * time.Hour

// defaultNamespace préfixe les clés du pilote redis.
const defaultNamespace = "idempotency"

// Module expose les ports de l'idempotence.
type Module struct {
	Reserve  ports.Reserve
	Complete ports.Complete
	Release  ports.Release
	Purge    ports.Purge
}

// Deps porte les dépendances que les pilotes peuvent réclamer.
//
// Pool et Cache peuvent être nil : le pilote `memory` n'a besoin d'aucun des
// deux, et c'est précisément ce qui permet de démarrer sans base et sans Redis.
type Deps struct {
	Pool  *pgxpool.Pool
	Cache *goredis.Client
	Now   func() time.Time
}

// ErrDisabled signale un appel à un module désactivé.
//
// Un module désactivé échoue explicitement plutôt que de laisser passer : une
// idempotence inerte qui « marche quand même » autoriserait les doublons sans
// jamais se signaler. C'est le pire défaut possible, parce qu'il est invisible
// jusqu'au jour où il coûte cher.
var ErrDisabled = errors.New("module idempotency désactivé dans config/modules.yaml")

// ErrPoolRequired signale un pilote qui exige une base absente.
var ErrPoolRequired = errors.New("le pilote postgres exige une connexion à la base")

// ErrCacheRequired signale un pilote qui exige un cache absent.
var ErrCacheRequired = errors.New("le pilote redis exige une connexion au cache")

var errUnknownDriver = errors.New("pilote idempotency inconnu")

// New construit le module selon la configuration.
//
// Un pilote inconnu refuse le démarrage : la validation de configuration l'a déjà
// rejeté, et ce second refus garantit qu'aucun chemin ne contourne le premier.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	ttl, err := cfg.DurationOption("ttl", defaultTTL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}

	switch cfg.Driver {
	case "memory":
		return fromMemory(memory.New(ttl, deps.Now)), nil
	case "postgres":
		if deps.Pool == nil {
			return Module{}, ErrPoolRequired
		}
		return fromPostgres(postgres.New(deps.Pool, ttl)), nil
	case "redis":
		return withRedis(cfg, deps, ttl)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// withRedis construit le pilote redis, dont les options sont plus riches.
func withRedis(cfg config.Module, deps Deps, ttl time.Duration) (Module, error) {
	if deps.Cache == nil {
		return Module{}, ErrCacheRequired
	}
	namespace, err := cfg.StringOption("namespace", defaultNamespace)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store := redis.New(deps.Cache, namespace, ttl)
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}, nil
}

// fromMemory assemble les ports du pilote mémoire.
func fromMemory(store *memory.Store) Module {
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}
}

// fromPostgres assemble les ports du pilote postgres.
func fromPostgres(store *postgres.Store) Module {
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}
}

// disabled retourne des ports qui refusent explicitement.
func disabled() Module {
	return Module{
		Reserve: func(context.Context, domain.Request) (domain.Reservation, error) {
			return domain.Reservation{}, ErrDisabled
		},
		Complete: func(context.Context, domain.Key, []byte) error { return ErrDisabled },
		Release:  func(context.Context, domain.Key) error { return ErrDisabled },
		Purge:    func(context.Context) (int64, error) { return 0, ErrDisabled },
	}
}
