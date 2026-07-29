package config

import "strings"

// Admitted event relays.
//
// These names are duplicated from `internal/infrastructure/messaging` and NOT
// imported: messaging depends on config, and the reverse would make a cycle.
// Validation is the only place where the duplication matters, and the addition
// of a relay that forgot this list would show up at startup, not in production.
const (
	relayInproc   = "inproc"
	relayKafka    = "kafka"
	relayRabbitMQ = "rabbitmq"
	relayNoop     = "noop"
)

// Messaging carries the event relay.
//
// The relay is INTERCHANGEABLE: the outbox guarantees durability upstream, so
// changing broker touches not one line of the core (ADR 010).
type Messaging struct {
	Driver         string   `yaml:"driver"`
	TopicPrefix    string   `yaml:"topic_prefix"`
	ConsumerGroup  string   `yaml:"consumer_group"`
	PublishTimeout Duration `yaml:"publish_timeout"`
	Kafka          Kafka    `yaml:"kafka"`
	RabbitMQ       RabbitMQ `yaml:"rabbitmq"`
}

// Kafka carries the parameters of the Kafka relay.
type Kafka struct {
	Brokers                []string `yaml:"brokers"`
	AllowAutoTopicCreation bool     `yaml:"allow_auto_topic_creation"`
}

// RabbitMQ carries the parameters of the AMQP relay.
type RabbitMQ struct {
	URL      string `yaml:"url"`
	Exchange string `yaml:"exchange"`
}

// Topic derives the destination name of an event type.
//
// The dot becomes a dash: AMQP gives the dot a hierarchical routing meaning,
// and Kafka reserves it for its metrics conventions.
func (m Messaging) Topic(eventType string) string {
	return m.TopicPrefix + "." + strings.ReplaceAll(eventType, ".", "-")
}
