// Package messaging abstracts the event transport between modules.
//
// # The outbox does not stand in place of the broker, it stands in front of it
//
// A module ALWAYS writes into the outbox, inside its business transaction: that
// is what guarantees no event is lost nor phantom (ADR 006). The worker then
// dispatches and hands the envelope over to a RELAY — it is the relay that
// knows Kafka, RabbitMQ, or nothing at all.
//
// Consequence: changing broker touches no line of `domain/`, `ports/` or
// `application/`. It is one line of configuration.
//
//	module ──► outbox (transactional) ──► worker ──► relay ──► Kafka / AMQP / inproc
//
// # Choosing the relay
//
//	messaging.driver: inproc     everything in the same process (default)
//	messaging.driver: kafka      messaging.kafka.brokers: [host:9092]
//	messaging.driver: rabbitmq   messaging.rabbitmq.url: amqp://…
//	messaging.driver: noop       no publication at all (tests, migrations)
//
// # One file per relay
//
// This file carries only the LANGUAGE — the envelope and the function types.
// Each relay lives in its own (rules/tests.md §2):
//
//	broker.go      Broker and New: the only point that chooses a transport
//	inproc.go      in-memory bus — the normal mode of a modular monolith
//	noop.go        no publication
//	kafka.go       Kafka relay       ⚠️ written, never run against a real broker
//	rabbitmq.go    AMQP relay        ⚠️ written, never run against a real broker
//	with_retry.go  retry decorator, shared by both network relays
//
// Both network relays are UNPROVEN. Separating them physically makes that
// boundary visible: a whole file is marked unproven, rather than a paragraph
// lost in the middle of code that does run.
package messaging

import (
	"context"
	"errors"
	"time"
)

// Driver names a relay.
type Driver string

// The available relays.
const (
	DriverInproc   Driver = "inproc"
	DriverKafka    Driver = "kafka"
	DriverRabbitMQ Driver = "rabbitmq"
	DriverNoop     Driver = "noop"
)

// ErrUnknownDriver signals an unrecognised relay.
//
// Deny by default: an unknown driver refuses to start rather than silently
// falling back on "no publication".
var ErrUnknownDriver = errors.New("unknown messaging relay")

// Envelope is the transport format of an event.
//
// It is deliberately poor and free of any domain Go type: that is what lets a
// consumer written in another language, or deployed separately, read it.
// Payload is opaque JSON.
type Envelope struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	AggregateID string            `json:"aggregate_id"`
	Payload     []byte            `json:"payload"`
	TraceParent string            `json:"traceparent,omitempty"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// Publisher hands an envelope over to the transport.
//
// A function type, hence substitutable by a closure in tests — and a decorator
// (retry, metric) is a plain `func(Publisher) Publisher`.
type Publisher = func(ctx context.Context, env Envelope) error

// Handler consumes an envelope.
//
// It MUST be idempotent: every transport here is "at least once".
type Handler = func(ctx context.Context, env Envelope) error

// Consumer subscribes to event types and loops until cancellation.
type Consumer interface {
	Subscribe(eventType string, handler Handler)
	Run(ctx context.Context) error
}

// Closer releases the transport's resources.
type Closer = func() error
