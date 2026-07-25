// Package ports déclare les contrats de la configuration dynamique.
//
// Ce paquet ne contient QUE des déclarations de types.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// IsEnabled répond à « ce drapeau est-il actif ? ».
//
// # Pourquoi ce port ne retourne PAS d'erreur
//
// C'est la seule exception à la règle « un module noyau retourne error », et elle
// est assumée. Le contrat du port EST le repli : drapeau inconnu, magasin
// injoignable, valeur illisible — la réponse est `false` dans les trois cas.
//
// Rendre `(bool, error)` obligerait chaque appel — il y en a un par branche de
// code masquée — à traiter une erreur dont la seule réponse correcte est déjà
// connue. En pratique, ces sites écriraient `enabled, _ := flag(ctx, key)`, et le
// repli deviendrait implicite au lieu d'être garanti par le port.
//
// La contrepartie est réelle et doit être connue : une panne du magasin ÉTEINT
// silencieusement les fonctionnalités masquées. Un pilote qui échoue doit donc le
// journaliser lui-même — c'est ce que le contrat lui impose en échange.
type IsEnabled = func(ctx context.Context, key domain.FlagKey) bool

// GetSetting lit un réglage métier. Une absence n'est pas une erreur : c'est le
// cas nominal d'un réglage qui n'a jamais été posé.
type GetSetting = func(ctx context.Context, key domain.SettingKey) domain.Setting

// Set écrit une valeur dynamique.
//
// Retourne `domain.ErrReadOnly` sur un magasin qui n'accepte pas l'écriture. Un
// pilote qui accepterait l'écriture sans la persister serait pire : l'appelant
// croirait avoir changé quelque chose.
type Set = func(ctx context.Context, change domain.Change) error

// Invalidate purge le cache local.
//
// Sans contexte ni erreur : purger un cache mémoire ne peut ni échouer, ni
// attendre. À appeler après une écriture faite par une AUTRE réplique.
type Invalidate = func()
