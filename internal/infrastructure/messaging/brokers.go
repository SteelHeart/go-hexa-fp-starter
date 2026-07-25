package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/segmentio/kafka-go"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// â”€â”€â”€ Kafka â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// newKafka construit le relais Kafka.
//
// âš ï¸ Ã‰CRIT, NON PROUVÃ‰ : aucune exÃ©cution contre un Kafka rÃ©el n'a eu lieu.
// Ne pas le prÃ©senter comme fonctionnel (rules/README.md Â§ rÃ¨gle d'or 2).
func newKafka(cfg config.Messaging, logger *slog.Logger) (Publisher, Consumer, Closer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		WriteTimeout: cfg.PublishTimeout.Duration(),
		// Allow n'Ã©crit pas les topics manquants en production : les crÃ©er Ã  la
		// volÃ©e masque une erreur de configuration. Vrai en dev uniquement.
		AllowAutoTopicCreation: cfg.Kafka.AllowAutoTopicCreation,
	}

	publish := func(ctx context.Context, env Envelope) error {
		raw, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("sÃ©rialisation de l'enveloppe: %w", err)
		}
		msg := kafka.Message{
			Topic: cfg.Topic(env.Type),
			// La clÃ© est l'agrÃ©gat : Kafka garantit l'ordre par partition, donc
			// tous les Ã©vÃ©nements d'un mÃªme agrÃ©gat restent ordonnÃ©s.
			Key:     []byte(env.AggregateID),
			Value:   raw,
			Headers: kafkaHeaders(env),
		}
		if err := writer.WriteMessages(ctx, msg); err != nil {
			return fmt.Errorf("publication Kafka sur %s: %w", msg.Topic, err)
		}
		return nil
	}

	consumer := &kafkaConsumer{cfg: cfg, logger: logger, handlers: map[string]Handler{}}
	closer := func() error {
		if err := writer.Close(); err != nil {
			return fmt.Errorf("fermeture du writer Kafka: %w", err)
		}
		return nil
	}
	return WithRetry(publish, 3, 200*time.Millisecond), consumer, closer, nil
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

// Run ouvre un lecteur par type d'Ã©vÃ©nement.
//
// Un lecteur par topic plutÃ´t qu'un lecteur multi-topics : le dÃ©calage
// (Â« offset Â») est commitÃ© par topic, donc un consommateur lent n'empÃªche pas
// les autres d'avancer.
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
			c.logger.ErrorContext(ctx, "lecture Kafka en Ã©chec",
				slog.String("event_type", eventType), slog.Any("error", err))
			continue
		}
		if err := c.handle(ctx, msg, handler); err != nil {
			// Pas de commit : le message sera relivrÃ©. Le consommateur Ã©tant
			// idempotent, c'est le comportement souhaitÃ©.
			c.logger.ErrorContext(ctx, "consommation en Ã©chec, message non commitÃ©",
				slog.String("event_type", eventType), slog.Any("error", err))
			continue
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "commit Kafka en Ã©chec", slog.Any("error", err))
		}
	}
}

func (c *kafkaConsumer) handle(ctx context.Context, msg kafka.Message, handler Handler) error {
	var env Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		// Message illisible : le rejouer ne servira Ã  rien. On le commite pour
		// ne pas bloquer la partition, mais on journalise en Error.
		c.logger.ErrorContext(ctx, "enveloppe Kafka illisible", slog.Any("error", err))
		return nil
	}
	return handler(ctx, env)
}

// â”€â”€â”€ RabbitMQ â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// newRabbitMQ construit le relais AMQP.
//
// âš ï¸ Ã‰CRIT, NON PROUVÃ‰ : aucune exÃ©cution contre un RabbitMQ rÃ©el n'a eu lieu.
func newRabbitMQ(cfg config.Messaging, logger *slog.Logger) (Publisher, Consumer, Closer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connexion AMQP: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("ouverture du canal AMQP: %w", err)
	}
	// Ã‰change durable de type topic : les consommateurs se lient par motif,
	// donc ajouter un consommateur ne touche pas le producteur.
	if err := channel.ExchangeDeclare(cfg.RabbitMQ.Exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("dÃ©claration de l'Ã©change: %w", err)
	}

	publish := func(ctx context.Context, env Envelope) error {
		raw, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("sÃ©rialisation de l'enveloppe: %w", err)
		}
		pubCtx, cancel := context.WithTimeout(ctx, cfg.PublishTimeout.Duration())
		defer cancel()
		err = channel.PublishWithContext(pubCtx, cfg.RabbitMQ.Exchange, cfg.Topic(env.Type), false, false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        raw,
				MessageId:   env.ID,
				Timestamp:   env.OccurredAt,
				// Persistant : sans cela, un redÃ©marrage du broker perd les
				// messages en file.
				DeliveryMode: amqp.Persistent,
				Headers:      amqp.Table{"traceparent": env.TraceParent},
			})
		if err != nil {
			return fmt.Errorf("publication AMQP: %w", err)
		}
		return nil
	}

	consumer := &amqpConsumer{cfg: cfg, channel: channel, logger: logger, handlers: map[string]Handler{}}
	closer := func() error {
		if err := conn.Close(); err != nil {
			return fmt.Errorf("fermeture AMQP: %w", err)
		}
		return nil
	}
	return WithRetry(publish, 3, 200*time.Millisecond), consumer, closer, nil
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
		return fmt.Errorf("abonnement AMQP: %w", errors.Join(setupErrs...))
	}
	wg.Wait()
	return nil
}

// bind dÃ©clare une file durable et nommÃ©e par groupe de consommateurs, pour que
// N rÃ©pliques se partagent la charge au lieu de la dupliquer.
func (c *amqpConsumer) bind(eventType string) (<-chan amqp.Delivery, error) {
	name := c.cfg.ConsumerGroup + "." + c.cfg.Topic(eventType)
	queue, err := c.channel.QueueDeclare(name, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("dÃ©claration de la file %s: %w", name, err)
	}
	if err := c.channel.QueueBind(queue.Name, c.cfg.Topic(eventType), c.cfg.RabbitMQ.Exchange, false, nil); err != nil {
		return nil, fmt.Errorf("liaison de la file %s: %w", name, err)
	}
	deliveries, err := c.channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("consommation de la file %s: %w", name, err)
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
				// Illisible : requeue=false, sinon la file boucle sur ce message.
				c.logger.ErrorContext(ctx, "enveloppe AMQP illisible", slog.Any("error", err))
				_ = delivery.Nack(false, false)
				continue
			}
			if err := handler(ctx, env); err != nil {
				c.logger.ErrorContext(ctx, "consommation AMQP en Ã©chec",
					slog.String("event_type", env.Type), slog.Any("error", err))
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}
}
