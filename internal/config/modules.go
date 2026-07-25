package config

import (
	"fmt"
	"slices"
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
// Deny par défaut : ce qui n'est pas listé refuse le démarrage. Une faute de
// frappe dans un nom de pilote ne doit jamais se résoudre en « le plus proche ».
//
//nolint:gochecknoglobals // table de référence immuable, lue en validation
var knownDrivers = map[string][]string{
	"outbox":       {"memory", "postgres"},
	"idempotency":  {"memory", "postgres", "redis"},
	"dynconf":      {"file", "postgres"},
	"audit":        {"log", "postgres"},
	"storage":      {"disk", "s3", "gcs", "azure-blob", "sftp"},
	"notification": {"log", "smtp", "mailjet", "sendgrid", "ses", "postmark", "mailgun", "resend", "brevo"},
	"search":       {"postgres-fts", "bleve", "meilisearch", "typesense", "opensearch", "elasticsearch", "algolia"},
	"payment":      {"log", "stripe", "adyen", "paypal", "mollie"},
	"secrets":      {"env", "file", "sops", "vault", "aws-secrets-manager", "gcp-secret-manager", "azure-key-vault"},
	"tenancy":      {"rls", "schema", "database"},
	"workflow":     {"builtin", "temporal"},
	"document":     {"html", "gotenberg", "weasyprint", "chromedp"},
	"i18n":         {"embedded", "file", "postgres"},
	"scheduler":    {"advisory-lock", "cron-inproc", "external"},
	"ratelimit":    {"memory", "redis", "postgres", "gateway"},
}

// defaultDrivers donne le pilote sans dépendance externe de chaque module.
//
//nolint:gochecknoglobals // table de référence immuable, lue en validation
var defaultDrivers = map[string]string{
	"outbox":       "memory",
	"idempotency":  "memory",
	"dynconf":      "file",
	"audit":        "log",
	"storage":      "disk",
	"notification": "log",
	"search":       "postgres-fts",
	"payment":      "log",
	"secrets":      "env",
	"tenancy":      "rls",
	"workflow":     "builtin",
	"document":     "html",
	"i18n":         "embedded",
	"scheduler":    "advisory-lock",
	"ratelimit":    "memory",
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

// RequiresDatabase indique si la configuration exige une connexion Postgres.
//
// C'est ce qui permet à un binaire de n'ouvrir le pool que s'il en a besoin —
// et donc de démarrer sans base quand tous les pilotes sont en mémoire.
func (m Modules) RequiresDatabase() bool {
	for name := range knownDrivers {
		if !m.IsEnabled(name) {
			continue
		}
		switch m.DriverOf(name) {
		case "postgres", "postgres-fts", "advisory-lock", "rls", "schema", "database":
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
