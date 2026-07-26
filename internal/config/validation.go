package config

import (
	"errors"
	"fmt"
	"slices"
)

// validateCore délègue à un vérificateur par GROUPE de configuration.
//
// Un seul bloc couvrant six groupes avait une complexité de 13 : personne ne
// relit ce genre de fonction, on y ajoute une branche. Découpé par groupe, chaque
// morceau tient sous les yeux et se déplace avec son groupe le jour où il bouge.
func (c Config) validateCore() []error {
	// Capacité indicative : une configuration valide n'en remplit aucune, une
	// configuration bâclée en remplit une poignée.
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
			"app.env=%q inconnu (attendu: development, test, uat, production)", c.App.Env)}
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
		problems = append(problems, errors.New("database.dsn est obligatoire"))
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
		problems = append(problems, fmt.Errorf("http.port=%d hors plage", c.HTTP.Port))
	}
	if c.HTTP.ReadTimeout <= 0 {
		problems = append(problems, errors.New(
			"http.read_timeout doit être > 0 : une connexion sans délai immobilise une goroutine"))
	}
	return problems
}

func (c Config) validateMessaging() []error {
	switch c.Messaging.Driver {
	case relayInproc, relayKafka, relayRabbitMQ, relayNoop:
		return nil
	default:
		return []error{fmt.Errorf(
			"messaging.driver=%q inconnu (attendu: %s, %s, %s, %s)",
			c.Messaging.Driver, relayInproc, relayKafka, relayRabbitMQ, relayNoop)}
	}
}

func (c Config) validateWorker() []error {
	if c.Worker.MaxAttempts < 1 {
		return []error{errors.New("worker.max_attempts doit être >= 1")}
	}
	return nil
}

func (c Config) validateI18n() []error {
	if !slices.Contains(c.I18n.SupportedLocales, c.I18n.DefaultLocale) {
		return []error{fmt.Errorf(
			"i18n.default_locale=%q absent de i18n.supported_locales", c.I18n.DefaultLocale)}
	}
	return nil
}
