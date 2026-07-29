package config

import (
	"errors"
	"fmt"
	"slices"
)

// validateCore delegates to one checker per configuration GROUP.
//
// A single block covering six groups had a complexity of 13: nobody re-reads
// that kind of function, one just adds a branch to it. Split by group, each
// piece fits under the eyes and moves with its group the day it moves.
func (c Config) validateCore() []error {
	// Indicative capacity: a valid configuration fills none of them, a sloppy
	// configuration fills a handful.
	problems := make([]error, 0, 8)
	problems = append(problems, c.validateApp()...)
	problems = append(problems, c.validateSecurity()...)
	problems = append(problems, c.validateDatabase()...)
	problems = append(problems, c.validateHTTP()...)
	problems = append(problems, c.validateMessaging()...)
	problems = append(problems, c.validateWorker()...)
	problems = append(problems, c.validateI18n()...)
	return problems
}

func (c Config) validateApp() []error {
	switch c.App.Env {
	case EnvDevelopment, EnvTest, EnvUAT, EnvProduction:
		return nil
	default:
		return []error{fmt.Errorf(
			"app.env=%q unknown (expected: development, test, uat, production)", c.App.Env)}
	}
}

func (c Config) validateSecurity() []error {
	if _, err := c.Security.DecodedEncryptionKey(); err != nil {
		return []error{err}
	}
	return nil
}

func (c Config) validateDatabase() []error {
	var problems []error
	if c.Database.DSN == "" {
		problems = append(problems, errors.New("database.dsn is mandatory"))
	}
	if c.Database.MinConns > c.Database.MaxConns {
		problems = append(problems, fmt.Errorf(
			"database.min_conns=%d > database.max_conns=%d", c.Database.MinConns, c.Database.MaxConns))
	}
	return problems
}

func (c Config) validateHTTP() []error {
	const maxPort = 65535
	var problems []error
	if c.HTTP.Port < 1 || c.HTTP.Port > maxPort {
		problems = append(problems, fmt.Errorf("http.port=%d out of range", c.HTTP.Port))
	}
	if c.HTTP.ReadTimeout <= 0 {
		problems = append(problems, errors.New(
			"http.read_timeout must be > 0: a connection without a timeout ties up a goroutine"))
	}
	return problems
}

func (c Config) validateMessaging() []error {
	switch c.Messaging.Driver {
	case relayInproc, relayKafka, relayRabbitMQ, relayNoop:
		return nil
	default:
		return []error{fmt.Errorf(
			"messaging.driver=%q unknown (expected: %s, %s, %s, %s)",
			c.Messaging.Driver, relayInproc, relayKafka, relayRabbitMQ, relayNoop)}
	}
}

func (c Config) validateWorker() []error {
	if c.Worker.MaxAttempts < 1 {
		return []error{errors.New("worker.max_attempts must be >= 1")}
	}
	return nil
}

func (c Config) validateI18n() []error {
	if !slices.Contains(c.I18n.SupportedLocales, c.I18n.DefaultLocale) {
		return []error{fmt.Errorf(
			"i18n.default_locale=%q absent from i18n.supported_locales", c.I18n.DefaultLocale)}
	}
	return nil
}
