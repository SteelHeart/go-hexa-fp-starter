package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/segmentio/kafka-go"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// newKafka builds the Kafka relay.
//
// ⚠️ WRITTEN, UNPROVEN: no run against a real Kafka has ever taken place.
// Do not present it as working (rules/README.md § golden rule 2).
func newKafka(cfg config.Messaging, logger *slog.Logger) (Broker, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		WriteTimeout: cfg.PublishTimeout.Duration(),
		// Allow does not write the missing topics in production: creating them
		// on the fly hides a configuration error. True in dev only.
		AllowAutoTopicCreation: cfg.Kafka.AllowAutoTopicCreation,
	}

	publish := func(ctx context.Context, env Envelope) error {
		raw, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("serialisation of the envelope: %w", err)
		}
		msg := kafka.Message{
			Topic: cfg.Topic(env.Type),
			// The key is the aggregate: Kafka guarantees ordering per
			// partition, so all the events of one aggregate stay ordered.
			Key:     []byte(env.AggregateID),
			Value:   raw,
			Headers: kafkaHeaders(env),
		}
		if err := writer.WriteMessages(ctx, msg); err != nil {
			return fmt.Errorf("publication to Kafka on %s: %w", msg.Topic, err)
		}
		return nil
	}

	closer := func() error {
		if err := writer.Close(); err != nil {
			return fmt.Errorf("closing of the Kafka writer: %w", err)
		}
		return nil
	}
	return Broker{
		Publish: WithRetry(publish, publishAttempts, publishBackoff),
		Consume: &kafkaConsumer{cfg: cfg, logger: logger, handlers: map[string]Handler{}},
		Close:   closer,
	}, nil
}

func kafkaHeaders(env Envelope) []kafka.Header {
	headers := []kafka.Header{{Key: "event-type", Value: []byte(env.Type)}}
	if env.TraceParent != "" {
		headers = append(headers, kafka.Header{Key: "traceparent", Value: []byte(env.TraceParent)})
	}
	for name, value := range env.Headers {
		headers = append(headers, kafka.Header{Key: name, Value: []byte(value)})
	}
	return headers
}

type kafkaConsumer struct {
	cfg      config.Messaging
	logger   *slog.Logger
	handlers map[string]Handler
}

func (c *kafkaConsumer) Subscribe(eventType string, handler Handler) {
	c.handlers[eventType] = handler
}

// Run opens one reader per event type.
//
// One reader per topic rather than one multi-topic reader: the offset is
// committed per topic, so a slow consumer does not prevent the others from
// moving forward.
func (c *kafkaConsumer) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	for eventType, handler := range c.handlers {
		wg.Add(1)
		go func(evType string, h Handler) {
			defer wg.Done()
			c.consume(ctx, evType, h)
		}(eventType, handler)
	}
	wg.Wait()
	return nil
}

func (c *kafkaConsumer) consume(ctx context.Context, eventType string, handler Handler) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: c.cfg.Kafka.Brokers,
		Topic:   c.cfg.Topic(eventType),
		GroupID: c.cfg.ConsumerGroup,
	})
	defer func() { _ = reader.Close() }()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.ErrorContext(ctx, "Kafka read failed",
				slog.String("event_type", eventType), slog.Any("error", err))
			continue
		}
		if err := c.handle(ctx, msg, handler); err != nil {
			// No commit: the message will be redelivered. The consumer being
			// idempotent, this is the desired behaviour.
			c.logger.ErrorContext(ctx, "consumption failed, message not committed",
				slog.String("event_type", eventType), slog.Any("error", err))
			continue
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "Kafka commit failed", slog.Any("error", err))
		}
	}
}

func (c *kafkaConsumer) handle(ctx context.Context, msg kafka.Message, handler Handler) error {
	var env Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		// Unreadable message: replaying it will serve no purpose. We commit it
		// so as not to block the partition, but we log at Error.
		c.logger.ErrorContext(ctx, "unreadable Kafka envelope", slog.Any("error", err))
		return nil
	}
	return handler(ctx, env)
}
