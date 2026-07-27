// Package messaging abstrait le transport d'événements entre modules.
//
// # L'outbox n'est pas à la place du broker, elle est devant lui
//
// Un module écrit TOUJOURS dans l'outbox, dans sa transaction métier : c'est
// ce qui garantit qu'aucun événement n'est perdu ni fantôme (ADR 006). Le
// worker dépile ensuite et remet l'enveloppe à un RELAIS — c'est le relais qui
// connaît Kafka, RabbitMQ, ou rien du tout.
//
// Conséquence : changer de broker ne touche aucune ligne de `domain/`, `ports/`
// ou `application/`. C'est une ligne de configuration.
//
//	module ──► outbox (transactionnel) ──► worker ──► relais ──► Kafka / AMQP / inproc
//
// # Choisir le relais
//
//	messaging.driver: inproc     tout dans le même processus (défaut)
//	messaging.driver: kafka      messaging.kafka.brokers: [host:9092]
//	messaging.driver: rabbitmq   messaging.rabbitmq.url: amqp://…
//	messaging.driver: noop       aucune publication (tests, migrations)
//
// # Un fichier par relais
//
// Ce fichier ne porte que le LANGAGE — l'enveloppe et les types fonction. Chaque
// relais vit dans le sien (rules/tests.md §2) :
//
//	broker.go      Broker et New : le seul point qui choisit un transport
//	inproc.go      bus en mémoire — le mode normal d'un monolithe modulaire
//	noop.go        aucune publication
//	kafka.go       relais Kafka        ⚠️ écrit, jamais exécuté contre un broker réel
//	rabbitmq.go    relais AMQP         ⚠️ écrit, jamais exécuté contre un broker réel
//	with_retry.go  décorateur de réessai, commun aux deux relais réseau
//
// Les deux relais réseau sont NON PROUVÉS. Les séparer physiquement rend cette
// frontière visible : un fichier entier est marqué non prouvé, plutôt qu'un
// paragraphe perdu au milieu de code qui, lui, tourne.
package messaging

import (
	"context"
	"errors"
	"time"
)

// Driver nomme un relais.
type Driver string

// Les relais disponibles.
const (
	DriverInproc   Driver = "inproc"
	DriverKafka    Driver = "kafka"
	DriverRabbitMQ Driver = "rabbitmq"
	DriverNoop     Driver = "noop"
)

// ErrUnknownDriver signale un relais non reconnu.
//
// Deny par défaut : un pilote inconnu refuse le démarrage plutôt que de se
// rabattre silencieusement sur « aucune publication ».
var ErrUnknownDriver = errors.New("relais de messagerie inconnu")

// Envelope est le format de transport d'un événement.
//
// Il est volontairement pauvre et sans type Go du domaine : c'est ce qui permet
// à un consommateur écrit dans un autre langage, ou déployé séparément, de le
// lire. Payload est du JSON opaque.
type Envelope struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	AggregateID string            `json:"aggregate_id"`
	Payload     []byte            `json:"payload"`
	TraceParent string            `json:"traceparent,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Publisher remet une enveloppe au transport.
//
// Type fonction, donc substituable par une closure en test — et un décorateur
// (retry, métrique) est un simple `func(Publisher) Publisher`.
type Publisher = func(ctx context.Context, env Envelope) error

// Handler consomme une enveloppe.
//
// Il DOIT être idempotent : tous les transports d'ici sont « au moins une fois ».
type Handler = func(ctx context.Context, env Envelope) error

// Consumer s'abonne à des types d'événements et boucle jusqu'à annulation.
type Consumer interface {
	Subscribe(eventType string, handler Handler)
	Run(ctx context.Context) error
}

// Closer libère les ressources du transport.
type Closer = func() error
