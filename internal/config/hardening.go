package config

import (
	"errors"
	"fmt"
	"strings"
)

// validateHardening porte les exigences qui ne s'appliquent qu'hors local.
//
// Deny par défaut : ce qui n'est pas explicitement sûr est refusé.
//
// Ce fichier existe séparément de validation.go pour une raison précise : ce sont
// les seules règles dont l'application DÉPEND de l'environnement. Les mêler aux
// autres, c'est risquer d'ajouter un jour une exigence de production dans un
// vérificateur qui tourne aussi en local — où elle serait alors contournée en
// l'affaiblissant pour tout le monde.
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
			"database.migration_dsn doit différer de database.dsn hors développement " +
				"(le rôle applicatif ne possède pas le schéma)")}
	}
	return nil
}

func (c Config) hardenOrigins() []error {
	if len(c.HTTP.AllowedOrigins) == 0 {
		return []error{errors.New("http.allowed_origins ne peut pas être vide hors développement")}
	}
	var problems []error
	for _, origin := range c.HTTP.AllowedOrigins {
		switch {
		case origin == "*":
			problems = append(problems, errors.New(
				"http.allowed_origins ne peut pas contenir '*' hors développement"))
		case strings.HasPrefix(origin, "http://"):
			problems = append(problems, fmt.Errorf(
				"origine non chiffrée interdite hors développement: %s", origin))
		case origin == "":
			problems = append(problems, errors.New(
				"http.allowed_origins contient une entrée vide (référence ${VAR} non résolue ?)"))
		}
	}
	return problems
}

func (c Config) hardenMessaging() []error {
	if c.Messaging.Driver == relayKafka && c.Messaging.Kafka.AllowAutoTopicCreation {
		return []error{errors.New(
			"messaging.kafka.allow_auto_topic_creation doit être false hors développement : " +
				"créer un topic à la volée masque une erreur de configuration")}
	}
	return nil
}

func (c Config) hardenTelemetry() []error {
	if !c.Telemetry.Enabled {
		return []error{errors.New(
			"telemetry.enabled doit être true hors développement : un service non observable n'est pas exploitable")}
	}
	return nil
}
