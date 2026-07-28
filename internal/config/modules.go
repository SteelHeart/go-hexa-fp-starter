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
	// Options porte le paramétrage propre au pilote.
	//
	// Les VALEURS restent non typées — le catalogue des pilotes évolue plus vite
	// que le socle, et chaque pilote interprète les siennes à sa construction.
	// Les CLÉS, elles, sont déclarées par le module dans son `catalog.go`, et
	// toute clé inconnue refuse le démarrage (#93).
	//
	// La nuance est le tout du sujet : sans elle, une faute de frappe rendait la
	// valeur par défaut, sans un mot.
	Options map[string]any `yaml:"options"`
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

// IntOption lit une option entière du pilote.
//
// Refuse zéro et les valeurs négatives : aucune option entière du socle n'a de
// sens à zéro — un lot de zéro message ou zéro tentative autorisée décrivent un
// composant qui ne fait rien, silencieusement.
func (m Module) IntOption(key string, fallback int) (int, error) {
	raw, found := m.Options[key]
	if !found || raw == nil {
		return fallback, nil
	}

	var value int
	switch typed := raw.(type) {
	case int:
		value = typed
	case int64:
		value = int(typed)
	default:
		return 0, fmt.Errorf("options.%s doit être un entier, reçu %T", key, raw)
	}

	if value <= 0 {
		return 0, fmt.Errorf("options.%s doit être strictement positif, reçu %d", key, value)
	}
	return value, nil
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

// Get retourne la configuration d'un module.
//
// Un module absent de la configuration est considéré comme DÉSACTIVÉ : on
// n'active jamais une capacité que personne n'a demandée.
//
// Le pilote rendu est celui qui est ÉCRIT dans la valeur. Le défaut du
// catalogue y est posé par Resolve, que Load appelle — de sorte que cet
// accesseur ne mente jamais sur ce qu'il a reçu.
func (m Modules) Get(name string) Module {
	return m[name]
}

// Resolve rend une copie où chaque module actif porte un pilote explicite.
//
// C'est une fonction PURE : elle ne modifie pas son entrée. Elle existe pour que
// « appliquer les défauts » soit une étape visible, au lieu d'un effet caché
// dans un accesseur — le reproche exact qu'on peut faire à un conteneur
// d'injection (ADR 004).
func (m Modules) Resolve(catalog ModuleCatalog) Modules {
	resolved := make(Modules, len(m))
	for name, mod := range m {
		if mod.Driver == "" {
			mod.Driver = catalog.DefaultDriver(name)
		}
		resolved[name] = mod
	}
	// Les modules du catalogue que la configuration ne mentionne PAS reçoivent
	// aussi leur pilote par défaut, désactivés.
	//
	// Sans ça, un module absent du fichier rendrait un pilote vide, et le
	// composition root — qui monte `outbox` inconditionnellement — le
	// construirait avec la chaîne vide, donc échouerait sur « pilote inconnu ».
	// Autrement dit : une configuration MINIMALE cesserait de démarrer, ce qui
	// contredirait frontalement la promesse de l'ADR 012.
	//
	// Ils restent désactivés : porter un pilote n'active rien.
	for name, set := range catalog {
		if _, declared := resolved[name]; declared {
			continue
		}
		resolved[name] = Module{Enabled: false, Driver: set.Default}
	}
	return resolved
}

// IsEnabled indique si un module est actif.
func (m Modules) IsEnabled(name string) bool { return m.Get(name).Enabled }

// DriverOf retourne le pilote retenu pour un module.
func (m Modules) DriverOf(name string) string { return m.Get(name).Driver }

// RequiresSQL indique si la configuration exige une base — sans présumer du MOTEUR.
//
// C'est ce qui permet à un binaire de n'ouvrir une connexion que s'il en a
// besoin, et donc de démarrer sans base quand tous les pilotes actifs vivent en
// mémoire ou sur fichier (ADR 012).
//
// Le catalogue est un PARAMÈTRE, pas un état caché : la réponse dépend des
// modules montés, et c'est le composition root qui les connaît (ADR 014). Un
// appelant qui oublie de le passer obtient `false`, pas un défaut arbitraire —
// aucun module monté, aucune ressource exigée.
func (m Modules) RequiresSQL(catalog ModuleCatalog) bool {
	return m.requires(catalog, func(r Resources) bool { return r.SQL })
}

// RequiresCache indique si la configuration exige un cache réseau.
func (m Modules) RequiresCache(catalog ModuleCatalog) bool {
	return m.requires(catalog, func(r Resources) bool { return r.Cache })
}

// requires factorise les deux questions ci-dessus.
//
// Elle n'itère que les modules DÉCLARÉS : un module absent de la configuration
// est désactivé (voir Get), donc il n'exige rien.
func (m Modules) requires(catalog ModuleCatalog, wanted func(Resources) bool) bool {
	for name := range m {
		if !m.IsEnabled(name) {
			continue
		}
		if wanted(catalog.Requires(name, m.DriverOf(name))) {
			return true
		}
	}
	return false
}

// validate vérifie que chaque module actif désigne un pilote connu du CATALOGUE.
//
// Un catalogue vide refuse tout, et c'est voulu : ce qui n'est pas monté n'est
// pas configurable. On ne configure pas ce qu'on n'a pas branché (ADR 014).
func (m Modules) validate(catalog ModuleCatalog) []error {
	var problems []error
	for name, mod := range m {
		allowed := catalog.AllowedDrivers(name)
		if allowed == nil {
			problems = append(problems, fmt.Errorf(
				"modules.%s : module inconnu — aucun module de ce nom n'est monté par le composition root", name))
			continue
		}
		if !mod.Enabled {
			continue
		}
		driver := mod.Driver
		if driver == "" {
			driver = catalog.DefaultDriver(name)
		}
		if !slices.Contains(allowed, driver) {
			problems = append(problems, fmt.Errorf(
				"modules.%s.driver=%q inconnu (attendu: %s)", name, driver, join(allowed)))
			continue
		}
		problems = append(problems, mod.unknownOptions(name, driver, catalog)...)
	}
	return problems
}

// unknownOptions refuse toute clé d'option que le pilote ne lit pas.
//
// # Le trou que cette fonction ferme
//
// Le deny-par-défaut s'arrêtait au nom du pilote. Ses OPTIONS passaient sans
// contrôle, parce que les accesseurs — `IntOption`, `DurationOption`… — rendent
// la valeur par défaut quand la clé est absente. Correct pris isolément : une
// option absente EST valide. Mais rien n'énumérait les clés connues, donc
// « absente » et « mal orthographiée » étaient indiscernables.
//
// Mesuré (#93) : `bath_size` au lieu de `batch_size` laissait le serveur
// démarrer, monter le module, et n'en rien dire. Le dépileur tournait avec un
// réglage que personne n'avait demandé — et sur `max_attempts` ou
// `base_backoff`, l'écart serait resté invisible et durable.
//
// Les clés admises sont déclarées par le module lui-même, dans son `catalog.go`,
// à côté du code qui les lit et en partageant ses constantes (ADR 014). Aucun
// fichier du framework ne les nomme.
func (m Module) unknownOptions(name, driver string, catalog ModuleCatalog) []error {
	admises := catalog.AllowedOptions(name, driver)

	// Trier les clés fautives : sans cela, l'ordre de map ferait varier le
	// message d'une exécution à l'autre, et deux traces cesseraient d'être
	// comparables.
	var fautives []string
	for key := range m.Options {
		if !slices.Contains(admises, key) {
			fautives = append(fautives, key)
		}
	}
	slices.Sort(fautives)

	// Un pilote sans aucune option admise mérite sa propre phrase : « admises: »
	// suivi du vide se lit comme un message tronqué, et envoie chercher un défaut
	// de l'outil plutôt que la faute de frappe.
	attendues := "ce pilote n'accepte aucune option"
	if len(admises) > 0 {
		attendues = "admises: " + join(admises)
	}

	problems := make([]error, 0, len(fautives))
	for _, key := range fautives {
		problems = append(problems, fmt.Errorf(
			"modules.%s.options.%s inconnue du pilote %q (%s)", name, key, driver, attendues))
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

// Modes de communication entre modules.
//
// Homonymie assumée avec relayInproc : ici « inproc » qualifie un APPEL direct
// entre deux modules du même binaire, là un relais d'événements.
const (
	transportInproc   = "inproc"
	transportHTTP     = "http"
	transportEvent    = "event"
	transportDisabled = "disabled"
)

// TransportFor résout le mode applicable à un module.
func (i Interop) TransportFor(module string) string {
	if raw, found := i.Transports[module]; found && raw != "" {
		return raw
	}
	if i.DefaultTransport == "" {
		return transportInproc
	}
	return i.DefaultTransport
}

// validate vérifie la cohérence des modes de communication.
func (i Interop) validate() []error {
	var problems []error
	allowed := []string{transportInproc, transportHTTP, transportEvent, transportDisabled}
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
