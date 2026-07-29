// Package config reads the startup configuration from the files in config/.
//
// Four principles, and they explain the whole package:
//
//  1. Files, not environment variables. The configuration is versioned, grouped
//     by domain, readable in review. Environment variables serve ONLY for
//     secrets, referenced by ${VAR} in the files.
//  2. Immutable — read ONCE at startup, passed by value. No access to
//     os.Getenv anywhere else in the repository.
//  3. Fail-fast — an invalid configuration refuses to start. A service that
//     starts half configured fails later, elsewhere, and for a reason that will
//     no longer have anything to do with it.
//  4. What changes without a redeployment is NOT here: business thresholds and
//     flags live in the core module internal/core/dynconf.
//
// # One file per group
//
// The physical split follows the split of config/ (rules/tests.md §2): one
// configuration group, one file, and it carries the type AND the methods that
// derive a value from it.
//
//	environment.go   the runtime environment and its predicates
//	http.go          HTTP server and rate limiting
//	database.go      database and cache
//	messaging.go     event relay
//	security.go      keys and hashing cost
//	observability.go logging, traces and metrics
//	groups.go        the groups without behaviour
//	modules.go       which modules are enabled, and on which driver
//	catalog.go       what a module DECLARES — the framework names no module
//	duration.go      durations, read as text and validated
//	loader.go        the merge of the layers and the substitution of the secrets
//	helpers.go       the helpers shared by the files above
//	validation.go    what makes a configuration invalid, everywhere
//	hardening.go     what makes a configuration invalid OUTSIDE local
//
// Validation stays grouped in its two files rather than scattered across each
// group: it is the only view from which one can answer "what refuses to
// start?" without opening ten files.
package config

import (
	"errors"
	"fmt"
)

// Config carries the whole startup configuration.
// One group = one file in conf/.
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

// App carries the identity of the service.
type App struct {
	Env     Environment `yaml:"env"`
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
}

// applyDefaults fills in the values the files might not carry.
//
// These are STRUCTURAL defaults, not business values: they guarantee that an
// incomplete configuration file does not produce a pool with zero connections.
func (c *Config) applyDefaults() {
	if c.App.Env == "" {
		c.App.Env = EnvDevelopment
	}
	if c.Database.MigrationDSN == "" {
		// Locally the two roles may coincide; validate() forbids it
		// elsewhere.
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

// validate gathers ALL the invalidities rather than stopping at the first one:
// fixing the configuration over six restarts is unacceptable.
func (c Config) validate(catalog ModuleCatalog) error {
	problems := make([]error, 0, 4)
	problems = append(problems, c.validateCore()...)
	problems = append(problems, c.validateHardening()...)
	problems = append(problems, c.Observability.validate()...)
	problems = append(problems, c.Modules.validate(catalog)...)
	problems = append(problems, c.Interop.validate()...)

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration: %w", errors.Join(problems...))
	}
	return nil
}
