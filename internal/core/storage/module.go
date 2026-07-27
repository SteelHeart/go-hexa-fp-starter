// Package storage est le module noyau de stockage d'objets.
//
// Composition root du module : le seul endroit qui connaît les pilotes.
//
// # Pourquoi un seul pilote livré
//
// Un pilote S3, GCS ou Azure Blob tire un SDK de plusieurs dizaines de mégaoctets
// dans le graphe de dépendances de TOUT projet engendré, y compris ceux qui
// n'iront jamais sur un magasin d'objets. Le règlement des pilotes tranche : les
// pilotes lourds sont des modules Go séparés (issue #22). Ils ne sont donc pas
// « oubliés », ils sont ailleurs — et `knownDrivers` ne liste que ce qui existe.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/drivers/disk"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/ports"
)

// Name est le nom du module dans config/modules.yaml.
const Name = "storage"

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
	driverDisk = "disk"
)

// Valeurs par défaut du pilote disk.
const (
	defaultBaseDir = "var/storage"
	defaultBaseURL = "/files"
)

// Module expose les ports du stockage.
type Module struct {
	Put    ports.Put
	Get    ports.Get
	Delete ports.Delete
}

// Deps porte les dépendances des pilotes.
//
// Vide aujourd'hui : le pilote `disk` ne réclame rien d'autre que sa
// configuration. Le type existe pour que l'ajout d'un pilote qui réclame un client
// ne change pas la signature de New — et donc aucun appelant.
type Deps struct{}

// ErrDisabled signale un appel à un module désactivé.
var ErrDisabled = errors.New("module storage désactivé dans config/modules.yaml")

var errUnknownDriver = errors.New("pilote storage inconnu")

// New construit le module selon la configuration.
func New(cfg config.Module, _ Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverDisk:
		return fromDisk(cfg)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// fromDisk construit le pilote local.
func fromDisk(cfg config.Module) (Module, error) {
	baseDir, err := cfg.StringOption("base_dir", defaultBaseDir)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	baseURL, err := cfg.StringOption("base_url", defaultBaseURL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store, err := disk.New(baseDir, baseURL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s: %w", Name, err)
	}
	return Module{Put: store.Put, Get: store.Get, Delete: store.Delete}, nil
}

// disabled retourne des ports qui refusent explicitement.
func disabled() Module {
	return Module{
		Put: func(context.Context, domain.Object) (domain.Located, error) {
			return domain.Located{}, ErrDisabled
		},
		Get:    func(context.Context, domain.Key) (io.ReadCloser, error) { return nil, ErrDisabled },
		Delete: func(context.Context, domain.Key) error { return ErrDisabled },
	}
}
