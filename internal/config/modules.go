package config

import (
	"fmt"
	"slices"
	"time"
)

// Modules porte l'activation et le choix de pilote de chaque module noyau.
//
// C'est le fichier qui rend vrai « hexa new puis go run, ça démarre » : les
// pilotes par défaut n'ont aucune dépendance externe
// ([ADR 012](documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
type Modules map[string]Module

// Module porte la configuration d'un module noyau.
type Module struct {
	// Enabled à false désactive le module : ses ports retournent une erreur
	// explicite plutôt que de se rabattre sur un comportement inerte. Un module
	// désactivé qui « marche quand même » est un piège.
	Enabled bool `yaml:"enabled"`
	// Driver nomme le pilote. Un pilote inconnu refuse le démarrage.
	Driver string `yaml:"driver"`
	// Options porte le paramétrage propre au pilote. Volontairement non typé :
	// le catalogue des pilotes évolue plus vite que le socle, et chaque pilote
	// valide ses propres options à sa construction.
	Options map[string]any `yaml:"options"`
}

// knownDrivers énumère les pilotes admis par module.
//
// # Cette table liste ce qui EXISTE, pas ce qui est prévu
//
// C'est une table de VALIDATION, pas un catalogue. Y faire figurer un pilote non
// construit produirait le pire des messages : la configuration accepterait la
// valeur, puis la fabrique du module refuserait au démarrage avec « pilote
// inconnu » — pour un pilote que la configuration venait de déclarer connu. Deux
// sources de vérité qui se contredisent valent moins qu'une seule qui refuse.
//
// Un module ABSENT de cette table ne peut pas être activé, et c'est voulu : on
// n'active pas un module dont le code n'existe pas.
//
// Le catalogue des pilotes ENVISAGÉS — une centaine — vit dans
// documentation/technique/pilotes.md. Un pilote y migre vers cette table le jour
// où il est écrit, testé, et où il documente ses NON-garanties.
//
// Deny par défaut : ce qui n'est pas listé refuse le démarrage. Une faute de
// frappe dans un nom de pilote ne doit jamais se résoudre en « le plus proche ».
//
//nolint:gochecknoglobals // table de référence immuable, lue en validation
var knownDrivers = map[string][]string{
	"outbox":      {"memory", "postgres"},
	"idempotency": {"memory", "postgres", "redis"},
	"dynconf":     {"file", "postgres"},
	"audit":       {"log", "postgres"},
	"storage":     {"disk"},
	"scheduler":   {"cron-inproc", "advisory-lock"},
}

// defaultDrivers donne le pilote sans dépendance externe de chaque module.
//
// C'est la table qui rend vraie la promesse « hexa new puis go run démarre » : le
// défaut n'est JAMAIS le pilote le plus complet, toujours celui qui n'exige rien.
//
//nolint:gochecknoglobals // table de référence immuable, lue en validation
var defaultDrivers = map[string]string{
	"outbox":      "memory",
	"idempotency": "memory",
	"dynconf":     "file",
	"audit":       "log",
	"storage":     "disk",
	// `advisory-lock` exigerait une base pour SIMPLEMENT répéter une tâche, y
	// compris dans un binaire mono-instance qui n'a personne avec qui s'accorder.
	"scheduler": "cron-inproc",
}

// DurationOption lit une option de durée du pilote.
//
// Les options ne sont pas typées à la lecture du fichier — le catalogue des
// pilotes évolue plus vite que le socle. Cet accesseur est donc le seul endroit
// où une durée d'option est interprétée, pour que tous les pilotes acceptent
// exactement la même écriture : `"24h"` ou un entier de secondes, comme le type
// Duration des champs typés.
//
// Une valeur présente mais illisible refuse le démarrage : se rabattre
// silencieusement sur la valeur par défaut donnerait un TTL surprise.
func (m Module) DurationOption(key string, fallback time.Duration) (time.Duration, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}

	var parsed time.Duration
	switch value := raw.(type) {
	case string:
		var err error
		if parsed, err = time.ParseDuration(value); err != nil {
			return 0, fmt.Errorf("options.%s=%q n'est pas une durée (ex: \"24h\"): %w", key, value, err)
		}
	case int:
		parsed = time.Duration(value) * time.Second
	case int64:
		parsed = time.Duration(value) * time.Second
	default:
		return 0, fmt.Errorf("options.%s doit être une durée ou un entier de secondes, reçu %T", key, raw)
	}

	if parsed <= 0 {
		return 0, fmt.Errorf("options.%s doit être strictement positive, reçu %v", key, parsed)
	}
	return parsed, nil
}

// MapOption lit un groupe d'options imbriqué.
//
// Un groupe absent rend une table vide et non une erreur : ne rien déclarer est
// une configuration valide. C'est une valeur du MAUVAIS type qui est refusée,
// parce qu'elle trahit une faute de frappe.
func (m Module) MapOption(key string) (map[string]any, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return map[string]any{}, nil
	}
	nested, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("options.%s doit être une table de valeurs, reçu %T", key, raw)
	}
	return nested, nil
}

// StringOption lit une option textuelle du pilote.
//
// Une valeur présente mais vide est refusée : elle trahit une variable
// d'environnement non substituée, pas une intention.
func (m Module) StringOption(key, fallback string) (string, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}
	text, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("options.%s doit être une chaîne, reçu %T", key, raw)
	}
	if text == "" {
		return "", fmt.Errorf("options.%s est présente mais vide", key)
	}
	return text, nil
}

// Get retourne la configuration d'un module, pilote par défaut compris.
//
// Un module absent de la configuration est considéré comme DÉSACTIVÉ : on
// n'active jamais une capacité que personne n'a demandée.
func (m Modules) Get(name string) Module {
	mod, found := m[name]
	if !found {
		return Module{Enabled: false, Driver: defaultDrivers[name]}
	}
	if mod.Driver == "" {
		mod.Driver = defaultDrivers[name]
	}
	return mod
}

// IsEnabled indique si un module est actif.
func (m Modules) IsEnabled(name string) bool { return m.Get(name).Enabled }

// DriverOf retourne le pilote retenu pour un module.
func (m Modules) DriverOf(name string) string { return m.Get(name).Driver }

// sqlBackedDrivers énumère les pilotes qui exigent une base SQL, quel que soit
// le MOTEUR.
//
// # Aucun moteur n'est imposé par le socle
//
// `postgres` figure ici comme un pilote parmi d'autres, pas comme une
// obligation. Le jour où un pilote `mysql`, `sqlite` ou `mssql` existe, il
// s'ajoute à cette table et rien d'autre ne change : les ports ne nomment aucun
// moteur, et chaque pilote possède son propre SQL et son propre client.
//
// C'est le sens de l'ADR 012 : le socle définit des contrats, pas une pile.
//
//nolint:gochecknoglobals // table de référence immuable
var sqlBackedDrivers = map[string]struct{}{
	"postgres": {},
	// Ces trois-là n'ont pas encore de pilote. Ils figurent ici pour que la
	// première implémentation n'ait RIEN à changer d'autre — et pour que le code
	// dise, dès maintenant, qu'aucun moteur n'est imposé.
	"mysql":  {},
	"sqlite": {},
	"mssql":  {},
	// Élection entre répliques par verrou consultatif.
	"advisory-lock": {},
}

// RequiresSQL indique si la configuration exige une base SQL — sans présumer
// du moteur.
//
// C'est ce qui permet à un binaire de n'ouvrir une connexion que s'il en a
// besoin, et donc de démarrer sans base quand tous les pilotes actifs sont en
// mémoire ou sur fichier.
func (m Modules) RequiresSQL() bool {
	for name := range knownDrivers {
		if !m.IsEnabled(name) {
			continue
		}
		if _, needsSQL := sqlBackedDrivers[m.DriverOf(name)]; needsSQL {
			return true
		}
	}
	return false
}

// RequiresCache indique si la configuration exige une connexion Redis.
func (m Modules) RequiresCache() bool {
	for name := range knownDrivers {
		if m.IsEnabled(name) && m.DriverOf(name) == "redis" {
			return true
		}
	}
	return false
}

// validate vérifie que chaque module actif désigne un pilote connu.
func (m Modules) validate() []error {
	var problems []error
	for name, mod := range m {
		allowed, known := knownDrivers[name]
		if !known {
			problems = append(problems, fmt.Errorf(
				"modules.%s : module inconnu (voir documentation/technique/pilotes.md)", name))
			continue
		}
		if !mod.Enabled {
			continue
		}
		driver := m.DriverOf(name)
		if !slices.Contains(allowed, driver) {
			problems = append(problems, fmt.Errorf(
				"modules.%s.driver=%q inconnu (attendu: %s)", name, driver, join(allowed)))
		}
	}
	return problems
}

// Interop porte les modes de communication ENTRE modules.
//
// Distinct de Modules : ici on décide comment deux modules se parlent, pas
// comment un module s'implémente. Un module n'accède JAMAIS aux tables d'un
// autre (ADR 011).
type Interop struct {
	DefaultTransport string            `yaml:"default_transport"`
	CallTimeout      Duration          `yaml:"call_timeout"`
	Transports       map[string]string `yaml:"transports"`
	BaseURLs         map[string]string `yaml:"base_urls"`
}

// TransportFor résout le mode applicable à un module.
func (i Interop) TransportFor(module string) string {
	if raw, found := i.Transports[module]; found && raw != "" {
		return raw
	}
	if i.DefaultTransport == "" {
		return "inproc"
	}
	return i.DefaultTransport
}

// validate vérifie la cohérence des modes de communication.
func (i Interop) validate() []error {
	var problems []error
	allowed := []string{"inproc", "http", "event", "disabled"}
	if !slices.Contains(allowed, i.TransportFor("")) {
		problems = append(problems, fmt.Errorf(
			"interop.default_transport=%q inconnu (attendu: %s)", i.DefaultTransport, join(allowed)))
	}
	for module, mode := range i.Transports {
		if !slices.Contains(allowed, mode) {
			problems = append(problems, fmt.Errorf(
				"interop.transports.%s=%q inconnu (attendu: %s)", module, mode, join(allowed)))
			continue
		}
		if mode == "http" && i.BaseURLs[module] == "" {
			problems = append(problems, fmt.Errorf(
				"interop.base_urls.%s est requis quand le transport est http", module))
		}
	}
	return problems
}
