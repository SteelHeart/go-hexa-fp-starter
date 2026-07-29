package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// newRabbitMQ builds the AMQP relay.
//
// ⚠️ WRITTEN, UNPROVEN: no run against a real RabbitMQ has ever taken place.
func newRabbitMQ(cfg config.Messaging, logger *slog.Logger) (Broker, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return Broker{}, fmt.Errorf("connection to AMQP: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return Broker{}, fmt.Errorf("opening of the AMQP channel: %w", err)
	}
	// Durable exchange of topic kind: the consumers bind by pattern, so adding
	// a consumer does not touch the producer.
	const (
		durable    = true
		autoDelete = false
		internal   = false
		noWait     = false
	)
	if declareErr := channel.ExchangeDeclare(cfg.RabbitMQ.Exchange, amqp.ExchangeTopic,
		durable, autoDelete, internal, noWait, nil); declareErr != nil {
		_ = conn.Close()
		return Broker{}, fmt.Errorf("declaration of the exchange: %w", declareErr)
	}

	publish := func(ctx context.Context, env Envelope) error {
		raw, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("serialisation of the envelope: %w", err)
		}
		pubCtx, cancel := context.WithTimeout(ctx, cfg.PublishTimeout.Duration())
		defer cancel()
		err = channel.PublishWithContext(pubCtx, cfg.RabbitMQ.Exchange, cfg.Topic(env.Type), false, false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        raw,
				MessageId:   env.ID,
				Timestamp:   env.OccurredAt,
				// Persistent: without this, a restart of the broker loses the
				// messages sitting in the queue.
				DeliveryMode: amqp.Persistent,
				Headers:      amqp.Table{"traceparent": env.TraceParent},
			})
		if err != nil {
			return fmt.Errorf("publication to AMQP: %w", err)
		}
		return nil
	}

	closer := func() error {
		if closeErr := conn.Close(); closeErr != nil {
			return fmt.Errorf("closing of AMQP: %w", closeErr)
		}
		return nil
	}
	return Broker{
		Publish: WithRetry(publish, publishAttempts, publishBackoff),
		Consume: &amqpConsumer{cfg: cfg, channel: channel, logger: logger, handlers: map[string]Handler{}},
		Close:   closer,
	}, nil
}

type amqpConsumer struct {
	cfg      config.Messaging
	channel  *amqp.Channel
	logger   *slog.Logger
	handlers map[string]Handler
}

func (c *amqpConsumer) Subscribe(eventType string, handler Handler) {
	c.handlers[eventType] = handler
}

func (c *amqpConsumer) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	var setupErrs []error
	for eventType, handler := range c.handlers {
		deliveries, err := c.bind(eventType)
		if err != nil {
			setupErrs = append(setupErrs, err)
			continue
		}
		wg.Add(1)
		go func(h Handler, in <-chan amqp.Delivery) {
			defer wg.Done()
			c.consume(ctx, h, in)
		}(handler, deliveries)
	}
	if len(setupErrs) > 0 {
		return fmt.Errorf("subscription to AMQP: %w", errors.Join(setupErrs...))
	}
	wg.Wait()
	return nil
}

// bind declares a queue that is durable and named after the consumer group, so
// that N replicas share the load instead of duplicating it.
func (c *amqpConsumer) bind(eventType string) (<-chan amqp.Delivery, error) {
	name := c.cfg.ConsumerGroup + "." + c.cfg.Topic(eventType)
	queue, err := c.channel.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("declaration of the queue %s: %w", name, err)
	}
	if bindErr := c.channel.QueueBind(queue.Name, c.cfg.Topic(eventType),
		c.cfg.RabbitMQ.Exchange, false, nil); bindErr != nil {
		return nil, fmt.Errorf("binding of the queue %s: %w", name, bindErr)
	}
	deliveries, err := c.channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consumption of the queue %s: %w", name, err)
	}
	return deliveries, nil
}

func (c *amqpConsumer) consume(ctx context.Context, handler Handler, in <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, open := <-in:
			if !open {
				return
			}
			var env Envelope
			if err := json.Unmarshal(delivery.Body, &env); err != nil {
				// Unreadable: requeue=false, otherwise the queue loops on this
				// message.
				c.logger.ErrorContext(ctx, "unreadable AMQP envelope", slog.Any("error", err))
				_ = delivery.Nack(false, false)
				continue
			}
			if err := handler(ctx, env); err != nil {
				c.logger.ErrorContext(ctx, "AMQP consumption failed",
					slog.String("event_type", env.Type), slog.Any("error", err))
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}
