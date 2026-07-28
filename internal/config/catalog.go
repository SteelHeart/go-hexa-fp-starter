package config

// Catalogue des modules — ADR 014.
//
// # Ce fichier définit des TYPES, jamais du contenu
//
// Aucun nom de module n'est écrit ici, et c'est la décision entière. Ce paquet
// portait deux tables — `knownDrivers` et `defaultDrivers` — qui énuméraient les
// six modules noyau. Un module absent était refusé.
//
// Excellent deny-par-défaut. Pour le socle. Pour une application, ça voulait
// dire : déclarer son propre module `billing` oblige à modifier un fichier du
// framework. Mesuré, pas déduit — le binaire répondait
// `modules.billing : module inconnu` avec un code de retour 1.
//
// La règle 7 d'`arch-go` interdit déjà à ce paquet de dépendre d'un paquet
// interne. Elle était respectée à la lettre et contournée dans l'esprit : le
// paquet ne dépendait d'aucun module, mais il les NOMMAIT. Le catalogue est
// désormais RECU, construit par le composition root, seul code autorisé à tout
// connaître (ADR 004).
//
// Un catalogue vide refuse TOUT. Le deny-par-défaut ne se relâche pas : il
// change de source.

import (
	"fmt"
	"slices"
	"sort"
)

// Resources dit ce qu'un pilote exige du processus qui l'héberge.
//
// C'est ce qui permet à un binaire de n'ouvrir une connexion que s'il en a
// besoin, et donc de démarrer sans base quand tous les pilotes actifs vivent en
// mémoire ou sur fichier — la promesse de l'ADR 012.
//
// La valeur zéro n'exige RIEN, et c'est le bon défaut : un pilote qui oublie de
// déclarer ses besoins n'ouvre aucune connexion, il échoue à sa construction.
// L'inverse ferait ouvrir une connexion inutile, silencieusement.
type Resources struct {
	// SQL : une base relationnelle, quel que soit le MOTEUR. Aucun moteur n'est
	// imposé par le socle (ADR 012) : `postgres` est un pilote parmi d'autres.
	SQL bool
	// Cache : un cache réseau partagé entre répliques.
	Cache bool
	// Options énumère les clés que ce pilote LIT dans `modules.<nom>.options`.
	//
	// Toute autre clé refuse le démarrage. Sans cette liste, « absente » et « mal
	// orthographiée » sont indiscernables : les accesseurs d'options rendent la
	// valeur par défaut dans les deux cas, et le pilote démarre avec un réglage
	// que personne n'a demandé.
	//
	// Mesuré avant d'être corrigé (#93) : `bath_size` au lieu de `batch_size`
	// laissait le serveur démarrer, monter le module et n'en rien dire.
	//
	// La valeur zéro n'admet AUCUNE option, et c'est le bon défaut : un pilote
	// qui oublie de déclarer les siennes voit sa configuration refusée, ce qui se
	// remarque. L'inverse rouvrirait le trou pour tous les pilotes à la fois.
	Options []string
}

// DriverSet déclare les pilotes d'UN module.
//
// `Drivers` porte à la fois la liste des pilotes admis et ce que chacun exige :
// une seule table, donc aucun risque de déclarer un pilote sans dire ce qu'il
// consomme, ni l'inverse.
type DriverSet struct {
	// Default est le pilote retenu quand la configuration n'en nomme aucun.
	//
	// Il DOIT être celui qui n'exige rien : c'est ce qui rend vraie la promesse
	// « `hexa new` puis `go run`, ça démarre ». Jamais le plus complet.
	Default string
	// Drivers énumère les pilotes admis et leurs besoins.
	Drivers map[string]Resources
}

// ModuleCatalog associe un nom de module à ses pilotes.
//
// Il est construit par le composition root, qui fusionne le catalogue de chaque
// module monté — noyau comme métier. Un module qui n'est pas monté n'y figure
// pas, donc n'est pas configurable : on ne configure pas ce qu'on n'a pas
// branché.
type ModuleCatalog map[string]DriverSet

// AllowedDrivers rend les pilotes admis d'un module, triés.
//
// Triés parce que cette liste sert à composer un message d'erreur : un ordre de
// map est aléatoire, et un message qui change à chaque exécution empêche de
// comparer deux traces.
func (c ModuleCatalog) AllowedDrivers(module string) []string {
	set, known := c[module]
	if !known {
		return nil
	}
	names := make([]string, 0, len(set.Drivers))
	for name := range set.Drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultDriver rend le pilote par défaut d'un module, ou la chaîne vide.
func (c ModuleCatalog) DefaultDriver(module string) string { return c[module].Default }

// AllowedOptions rend les clés d'options admises par un pilote, triées.
//
// Triées pour la même raison qu'AllowedDrivers : cette liste compose un message
// d'erreur, et un ordre de map change à chaque exécution — ce qui empêche de
// comparer deux traces.
func (c ModuleCatalog) AllowedOptions(module, driver string) []string {
	admises := slices.Clone(c.Requires(module, driver).Options)
	slices.Sort(admises)
	return admises
}

// Requires rend les besoins d'un pilote d'un module.
//
// Un pilote inconnu n'exige rien : c'est la validation qui le refuse, pas cet
// accesseur. Rendre `SQL: true` pour un pilote inexistant ferait ouvrir une
// connexion au nom d'un pilote qu'on va justement refuser.
func (c ModuleCatalog) Requires(module, driver string) Resources {
	return c[module].Drivers[driver]
}

// MergeCatalogs assemble les catalogues des modules montés.
//
// Deux modules ne peuvent pas revendiquer le même nom : ce serait une collision
// silencieuse, où l'un des deux se retrouverait configuré par les pilotes de
// l'autre. Refus explicite — deny par défaut.
//
// C'est le composition root qui appelle ceci, et lui seul : c'est le seul code
// qui connaît la liste des modules montés (ADR 004).
func MergeCatalogs(catalogs ...ModuleCatalog) (ModuleCatalog, error) {
	merged := ModuleCatalog{}
	for _, catalog := range catalogs {
		for name, set := range catalog {
			if _, collision := merged[name]; collision {
				return nil, fmt.Errorf(
					"catalogue: le module %q est déclaré deux fois — deux modules ne peuvent pas porter le même nom", name)
			}
			merged[name] = set
		}
	}
	return merged, nil
}
