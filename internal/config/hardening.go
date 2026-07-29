package config

import (
	"errors"
	"fmt"
	"strings"
)

// validateHardening carries the requirements that only apply outside local.
//
// Deny by default: what is not explicitly safe is refused.
//
// This file exists separately from validation.go for a precise reason: these
// are the only rules whose application DEPENDS on the environment. Mixing them
// with the others means risking, one day, adding a production requirement to a
// checker that also runs locally — where it would then be circumvented by
// weakening it for everybody.
func (c Config) validateHardening() []error {
	if c.App.Env.IsLocal() {
		return nil
	}
	problems := make([]error, 0, 8)
	problems = append(problems, c.hardenDatabase()...)
	problems = append(problems, c.hardenOrigins()...)
	problems = append(problems, c.hardenMessaging()...)
	problems = append(problems, c.hardenTelemetry()...)
	problems = append(problems, c.Observability.hardened()...)
	return problems
}

func (c Config) hardenDatabase() []error {
	if c.Database.MigrationDSN == c.Database.DSN {
		return []error{errors.New(
			"database.migration_dsn must differ from database.dsn outside development " +
				"(the application role does not own the schema)")}
	}
	return nil
}

func (c Config) hardenOrigins() []error {
	if len(c.HTTP.AllowedOrigins) == 0 {
		return []error{errors.New("http.allowed_origins cannot be empty outside development")}
	}
	var problems []error
	for _, origin := range c.HTTP.AllowedOrigins {
		switch {
		case origin == "*":
			problems = append(problems, errors.New(
				"http.allowed_origins cannot contain '*' outside development"))
		case strings.HasPrefix(origin, "http://"):
			problems = append(problems, fmt.Errorf(
				"unencrypted origin forbidden outside development: %s", origin))
		case origin == "":
			problems = append(problems, errors.New(
				"http.allowed_origins contains an empty entry (unresolved ${VAR} reference?)"))
		}
	}
	return problems
}

func (c Config) hardenMessaging() []error {
	if c.Messaging.Driver == relayKafka && c.Messaging.Kafka.AllowAutoTopicCreation {
		return []error{errors.New(
			"messaging.kafka.allow_auto_topic_creation must be false outside development: " +
				"creating a topic on the fly hides a configuration error")}
	}
	return nil
}

func (c Config) hardenTelemetry() []error {
	if !c.Telemetry.Enabled {
		return []error{errors.New(
			"telemetry.enabled must be true outside development: an unobservable service is not operable")}
	}
	return nil
}
