// Package dynconf est le module noyau de la configuration modifiable à chaud.
//
// Composition root du module : le seul endroit qui connaît les pilotes.
package dynconf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/file"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "dynconf"

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
	driverFile     = "file"
	driverPostgres = "postgres"
)

// defaultTTL borne la fraîcheur du cache du pilote postgres.
//
// Trente secondes est le compromis : assez court pour qu'un drapeau éteint en
// urgence le soit partout vite, assez long pour qu'une évaluation en boucle ne
// martèle pas la base.
const defaultTTL = 30 * time.Second

// Module expose les ports de la configuration dynamique.
type Module struct {
	IsEnabled  ports.IsEnabled
	GetSetting ports.GetSetting
	Set        ports.Set
	Invalidate ports.Invalidate
}

// Deps porte les dépendances que les pilotes peuvent réclamer.
//
// Toutes peuvent être nil : le pilote `file` n'en réclame aucune.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// ErrDisabled signale une écriture sur un module désactivé.
var ErrDisabled = errors.New("module dynconf désactivé dans config/modules.yaml")

// ErrPoolRequired signale un pilote qui exige une base absente.
var ErrPoolRequired = errors.New("le pilote postgres exige une connexion à la base")

// ErrLoggerRequired signale un pilote qui exige un journal absent.
//
// Le pilote postgres ne peut PAS retourner ses pannes : le contrat de
// ports.IsEnabled l'interdit. Il doit donc pouvoir les journaliser, sinon une base
// injoignable éteindrait les fonctionnalités masquées sans laisser de trace.
var ErrLoggerRequired = errors.New("le pilote postgres exige un journal")

var errUnknownDriver = errors.New("pilote dynconf inconnu")

// New construit le module selon la configuration.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverFile:
		return fromFile(cfg)
	case driverPostgres:
		return fromPostgres(cfg, deps)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// fromFile construit le pilote des valeurs versionnées.
func fromFile(cfg config.Module) (Module, error) {
	flags, err := cfg.MapOption("flags")
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	settings, err := cfg.MapOption("settings")
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store, err := file.New(flags, settings)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	return Module{
		IsEnabled:  store.Flag,
		GetSetting: store.Setting,
		Set:        store.Set,
		Invalidate: store.Invalidate,
	}, nil
}

// fromPostgres construit le pilote modifiable à chaud.
func fromPostgres(cfg config.Module, deps Deps) (Module, error) {
	if deps.Pool == nil {
		return Module{}, ErrPoolRequired
	}
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}
	ttl, err := cfg.DurationOption("ttl", defaultTTL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store := postgres.New(deps.Pool, deps.Logger, ttl, deps.Now)
	return Module{
		IsEnabled:  store.Flag,
		GetSetting: store.Setting,
		Set:        store.Set,
		Invalidate: store.Invalidate,
	}, nil
}

// disabled retourne des ports inertes en LECTURE et refusants en ÉCRITURE.
//
// # Pourquoi la lecture ne refuse pas ici, contrairement aux autres modules
//
// `ports.IsEnabled` ne peut pas retourner d'erreur, par contrat. La seule réponse
// possible est donc `false` — et c'est justement la réponse « deny par défaut »,
// celle qui éteint les fonctionnalités masquées plutôt que de les allumer.
// L'inertie coïncide ici avec le refus, ce qui n'est pas le cas ailleurs.
//
// `Set`, lui, PEUT parler : il refuse bruyamment. Un appelant qui croirait avoir
// changé un drapeau sur un module éteint est le seul vrai piège de ce module.
func disabled() Module {
	return Module{
		IsEnabled:  func(context.Context, domain.FlagKey) bool { return false },
		GetSetting: func(context.Context, domain.SettingKey) domain.Setting { return domain.Setting{} },
		Set:        func(context.Context, domain.Change) error { return ErrDisabled },
		Invalidate: func() {},
	}
}
