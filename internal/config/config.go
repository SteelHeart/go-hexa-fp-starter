// Package config lit la configuration de démarrage depuis les fichiers de conf/.
//
// Quatre principes, et ils expliquent tout le paquet :
//
//  1. Fichiers, pas variables d'environnement. La configuration est versionnée,
//     groupée par domaine, relisible en revue. Les variables d'environnement ne
//     servent QU'aux secrets, référencés par ${VAR} dans les fichiers.
//  2. Immuable — lue UNE fois au démarrage, passée par valeur. Aucun accès à
//     os.Getenv ailleurs dans le dépôt.
//  3. Fail-fast — une configuration invalide refuse le démarrage. Un service qui
//     démarre à moitié configuré échoue plus tard, ailleurs, et pour une raison
//     qui n'aura plus rien à voir.
//  4. Ce qui change sans redéploiement n'est PAS ici : les seuils métier et les
//     drapeaux vivent en base (internal/infrastructure/dynconf).
//
// # Un fichier par groupe
//
// Le découpage physique suit le découpage de conf/ (rules/tests.md §2) : un
// groupe de configuration, un fichier, et il porte le type ET les méthodes qui
// en dérivent une valeur.
//
//	environment.go  l'environnement d'exécution et ses prédicats
//	http.go         serveur HTTP et limitation de débit
//	database.go     base de données et cache
//	messaging.go    relais d'événements
//	security.go     clés et coût du hachage
//	groups.go       les groupes sans comportement
//	validation.go   ce qui rend une configuration invalide, partout
//	hardening.go    ce qui rend une configuration invalide HORS local
//
// La validation reste groupée dans ses deux fichiers plutôt que dispersée dans
// chaque groupe : c'est la seule vue d'où l'on peut répondre à « qu'est-ce qui
// refuse le démarrage ? » sans ouvrir dix fichiers.
package config

import (
	"errors"
	"fmt"
)

// Config porte l'intégralité de la configuration de démarrage.
// Un groupe = un fichier dans conf/.
type Config struct {
	App           App           `yaml:"app"`
	HTTP          HTTP          `yaml:"http"`
	Limits        Limits        `yaml:"limits"`
	Database      DB            `yaml:"database"`
	Cache         Cache         `yaml:"cache"`
	DynConf       DynConf       `yaml:"dynconf"`
	Worker        Worker        `yaml:"worker"`
	Storage       Storage       `yaml:"storage"`
	Messaging     Messaging     `yaml:"messaging"`
	Modules       Modules       `yaml:"modules"`
	Interop       Interop       `yaml:"interop"`
	Security      Security      `yaml:"security"`
	Mail          Mail          `yaml:"mail"`
	Telemetry     Telemetry     `yaml:"telemetry"`
	I18n          I18n          `yaml:"i18n"`
	Observability Observability `yaml:"observability"`
}

// App porte l'identité du service.
type App struct {
	Env     Environment `yaml:"env"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
}

// applyDefaults comble les valeurs que les fichiers pourraient ne pas porter.
//
// Ce sont des défauts STRUCTURELS, pas des valeurs métier : ils garantissent
// qu'un fichier de conf incomplet ne produit pas un pool à zéro connexion.
func (c *Config) applyDefaults() {
	if c.App.Env == "" {
		c.App.Env = EnvDevelopment
	}
	if c.Database.MigrationDSN == "" {
		// En local, les deux rôles peuvent coïncider ; validate() l'interdit
		// ailleurs.
		c.Database.MigrationDSN = c.Database.DSN
	}
	if c.Messaging.Driver == "" {
		c.Messaging.Driver = relayInproc
	}
	if c.Interop.DefaultTransport == "" {
		c.Interop.DefaultTransport = transportInproc
	}
	if c.Interop.Transports == nil {
		c.Interop.Transports = map[string]string{}
	}
	if c.Interop.BaseURLs == nil {
		c.Interop.BaseURLs = map[string]string{}
	}
	c.Observability.applyDefaults()
	if c.I18n.DefaultLocale == "" {
		c.I18n.DefaultLocale = "fr"
	}
	if len(c.I18n.SupportedLocales) == 0 {
		c.I18n.SupportedLocales = []string{c.I18n.DefaultLocale}
	}
}

// validate rassemble TOUTES les invalidités plutôt que de s'arrêter à la
// première : corriger la configuration en six redémarrages est inacceptable.
func (c Config) validate() error {
	problems := make([]error, 0, 4)
	problems = append(problems, c.validateCore()...)
	problems = append(problems, c.validateHardening()...)
	problems = append(problems, c.Observability.validate()...)
	problems = append(problems, c.Modules.validate()...)
	problems = append(problems, c.Interop.validate()...)

	if len(problems) > 0 {
		return fmt.Errorf("configuration invalide: %w", errors.Join(problems...))
	}
	return nil
}
