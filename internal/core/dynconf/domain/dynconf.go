// Package domain porte le vocabulaire de la configuration dynamique, sans
// dépendance.
//
// # La ligne de partage avec le paquet config
//
// Ce qui exige un redéploiement pour changer — DSN, ports, délais — est dans
// `config`. Ce qui doit pouvoir changer À CHAUD est ici : drapeaux de
// fonctionnalité et réglages métier.
//
// Les drapeaux ne sont pas un luxe : le tronc unique (ADR 007) impose que le
// travail incomplet passe derrière un drapeau plutôt que derrière une branche
// longue. Sans cette brique, la règle est intenable.
package domain

import (
	"errors"
	"fmt"
	"strconv"
)

// Kind sépare les deux natures de valeur dynamique.
//
// Séparées et non fondues dans un espace de noms unique : un drapeau se lit en
// booléen avec un repli sur `false`, un réglage se lit en texte avec une absence
// possible. Les confondre ferait passer un réglage vide pour un drapeau inactif.
type Kind string

const (
	// KindFlag désigne un drapeau de fonctionnalité.
	KindFlag Kind = "flag"
	// KindSetting désigne un réglage métier.
	KindSetting Kind = "setting"
)

// FlagKey identifie un drapeau de fonctionnalité.
type FlagKey string

// SettingKey identifie un réglage métier.
type SettingKey string

// Setting est le résultat de la lecture d'un réglage.
//
// `Found` distingue « absent » de « présent et vide » — deux situations que rien
// ne doit confondre : la première appelle une valeur par défaut côté appelant, la
// seconde est une décision d'exploitation.
type Setting struct {
	Value string
	Found bool
}

// ErrReadOnly signale un magasin qui n'accepte pas d'écriture.
//
// Le pilote `file` lit des valeurs VERSIONNÉES : les réécrire à l'exécution
// produirait une divergence entre le dépôt et ce qui tourne, invisible au
// prochain déploiement qui les écraserait.
var ErrReadOnly = errors.New("magasin de configuration dynamique en lecture seule")

// ErrInvalidChange refuse une écriture incomplète ou de nature inconnue.
var ErrInvalidChange = errors.New("modification de configuration dynamique invalide")

// Change est une écriture de valeur dynamique.
type Change struct {
	Kind  Kind
	Key   string
	Value string
}

// IsValid indique si la modification est exploitable.
//
// La valeur peut être vide — c'est une valeur légitime. La clé et la nature, non.
func (c Change) IsValid() bool {
	return c.Key != "" && (c.Kind == KindFlag || c.Kind == KindSetting)
}

// Qualify rend l'identifiant complet d'une valeur, nature comprise.
//
// Utilisé comme clé de cache et comme clé de recherche par tous les pilotes :
// deux pilotes qui ne qualifieraient pas pareil ne seraient pas substituables.
func Qualify(kind Kind, key string) string { return string(kind) + "." + key }

// ParseFlag interprète une valeur textuelle en booléen.
//
// # Deny par défaut
//
// Une valeur illisible donne `false`. Un drapeau qui s'activerait sur une valeur
// qu'on ne comprend pas serait exactement l'inverse de ce qu'on veut : la
// fonctionnalité incomplète qu'il masque partirait en production sur une faute de
// frappe.
//
// C'est le seul endroit où cette interprétation existe, pour que tous les pilotes
// répondent identiquement à `"1"`, `"true"` ou `"TRUE"`.
func ParseFlag(raw string) bool {
	enabled, err := strconv.ParseBool(raw)
	return err == nil && enabled
}

// FormatFlag rend la forme canonique d'un drapeau, pour l'écriture.
func FormatFlag(enabled bool) string { return strconv.FormatBool(enabled) }

// Describe rend une modification lisible dans un message d'erreur.
func (c Change) Describe() string { return fmt.Sprintf("%s.%s", c.Kind, c.Key) }
