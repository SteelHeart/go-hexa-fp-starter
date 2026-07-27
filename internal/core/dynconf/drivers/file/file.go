// Package file implémente la configuration dynamique depuis les valeurs
// versionnées du dépôt.
//
// Les valeurs viennent des `options` du module dans `config/modules.yaml`, et non
// d'un fichier à part : le chargeur refuse toute clé inconnue dans `config/*.yaml`
// (`KnownFields(true)`), donc un nouveau fichier casserait le démarrage sans
// modifier le type `Config`. Les options, elles, sont volontairement non typées.
//
// # NON-GARANTIES — à lire avant de l'utiliser
//
//   - **PAS modifiable à chaud.** C'est LA non-garantie du pilote, et elle
//     contredit la raison d'être du module. Changer un drapeau exige un
//     redéploiement. En développement, `config/local.yaml` — non versionné —
//     surcharge n'importe quelle valeur par fusion, sans toucher au dépôt.
//   - **`Set` refuse toujours** (`domain.ErrReadOnly`). Écrire dans un fichier
//     versionné à l'exécution créerait une divergence entre le dépôt et ce qui
//     tourne, que le prochain déploiement écraserait sans prévenir.
//
// Convient en développement, en test, et en production tant que les drapeaux
// changent au rythme des déploiements. Dès qu'il faut éteindre une fonctionnalité
// sans redéployer, passer au pilote `postgres`.
package file

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// Store est le magasin en lecture seule.
//
// Aucun verrou : la carte est construite une fois puis jamais modifiée. C'est
// l'immuabilité qui rend le magasin utilisable depuis plusieurs goroutines, pas
// un mutex.
type Store struct {
	values map[string]string
}

// New construit le magasin depuis les options du pilote.
//
// Une valeur non scalaire refuse le démarrage : `flags: {a: {b: 1}}` est une faute
// de frappe, pas une intention, et la laisser passer donnerait un drapeau
// silencieusement inactif.
func New(flags, settings map[string]any) (*Store, error) {
	values := make(map[string]string, len(flags)+len(settings))
	if err := fill(values, domain.KindFlag, "flags", flags); err != nil {
		return nil, err
	}
	if err := fill(values, domain.KindSetting, "settings", settings); err != nil {
		return nil, err
	}
	return &Store{values: values}, nil
}

// fill convertit un groupe d'options en valeurs textuelles qualifiées.
func fill(dst map[string]string, kind domain.Kind, label string, src map[string]any) error {
	for key, raw := range src {
		text, err := scalar(raw)
		if err != nil {
			return fmt.Errorf("options.%s.%s: %w", label, key, err)
		}
		dst[domain.Qualify(kind, key)] = text
	}
	return nil
}

// errNotScalar refuse une valeur qui n'est pas une valeur.
var errNotScalar = errors.New("valeur non scalaire")

// scalar rend la forme textuelle d'une option.
//
// Tout passe par du texte : c'est ce qui permet au pilote `postgres`, qui lit une
// colonne `text`, d'être substituable à celui-ci sans que l'appelant le sache.
func scalar(raw any) (string, error) {
	switch value := raw.(type) {
	case string:
		return value, nil
	case bool:
		return strconv.FormatBool(value), nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("%w (%T)", errNotScalar, raw)
	}
}

// Flag implémente ports.IsEnabled.
func (s *Store) Flag(_ context.Context, key domain.FlagKey) bool {
	raw, found := s.values[domain.Qualify(domain.KindFlag, string(key))]
	if !found {
		return false
	}
	return domain.ParseFlag(raw)
}

// Setting implémente ports.GetSetting.
func (s *Store) Setting(_ context.Context, key domain.SettingKey) domain.Setting {
	raw, found := s.values[domain.Qualify(domain.KindSetting, string(key))]
	return domain.Setting{Value: raw, Found: found}
}

// Set implémente ports.Set en refusant.
//
// Vérifie tout de même la validité de la modification : un appelant qui corrigera
// son pilote doit d'abord apprendre que sa modification était mal formée.
func (s *Store) Set(_ context.Context, change domain.Change) error {
	if !change.IsValid() {
		return fmt.Errorf("%w: %s", domain.ErrInvalidChange, change.Describe())
	}
	return fmt.Errorf("%w: %s", domain.ErrReadOnly, change.Describe())
}

// Invalidate implémente ports.Invalidate. Sans effet : rien n'est mis en cache,
// puisque tout est déjà en mémoire et immuable.
func (s *Store) Invalidate() {}
