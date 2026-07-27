package config

import "strings"

// Relais d'événements admis.
//
// Ces noms sont dupliqués depuis `internal/infrastructure/messaging` et NON
// importés : messaging dépend de config, l'inverse ferait un cycle. La validation
// est le seul point où la duplication compte, et un ajout de relais qui
// oublierait cette liste se verrait au démarrage, pas en production.
const (
	relayInproc   = "inproc"
	relayKafka    = "kafka"
	relayRabbitMQ = "rabbitmq"
	relayNoop     = "noop"
)

// Messaging porte le relais d'événements.
//
// Le relais est INTERCHANGEABLE : l'outbox garantit la durabilité en amont, donc
// changer de broker ne touche aucune ligne du cœur (ADR 010).
type Messaging struct {
	Driver         string   `yaml:"driver"`
	TopicPrefix    string   `yaml:"topic_prefix"`
	ConsumerGroup  string   `yaml:"consumer_group"`
	PublishTimeout Duration `yaml:"publish_timeout"`
	Kafka          Kafka    `yaml:"kafka"`
	RabbitMQ       RabbitMQ `yaml:"rabbitmq"`
}

// Kafka porte les paramètres du relais Kafka.
type Kafka struct {
	Brokers                []string `yaml:"brokers"`
	AllowAutoTopicCreation bool     `yaml:"allow_auto_topic_creation"`
}

// RabbitMQ porte les paramètres du relais AMQP.
type RabbitMQ struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
}

// Topic dérive le nom de destination d'un type d'événement.
//
// Le point devient un tiret : AMQP donne au point un sens de routage
// hiérarchique, et Kafka le réserve à ses conventions de métriques.
func (m Messaging) Topic(eventType string) string {
	return m.TopicPrefix + "." + strings.ReplaceAll(eventType, ".", "-")
}
