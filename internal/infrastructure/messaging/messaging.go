// Package messaging abstrait le transport d'événements entre modules.
//
// # L'outbox n'est pas à la place du broker, elle est devant lui
//
// Une feature écrit TOUJOURS dans l'outbox, dans sa transaction métier : c'est
// ce qui garantit qu'aucun événement n'est perdu ni fantôme (ADR 006). Le
// worker dépile ensuite et remet l'enveloppe à un RELAIS — c'est le relais qui
// connaît Kafka, RabbitMQ, ou rien du tout.
//
// Conséquence : changer de broker ne touche aucune ligne de `domain/`, `ports/`
// ou `application/`. C'est une variable d'environnement.
//
//	feature ──► outbox (Postgres, transactionnel)  ──► worker ──► relais ──► Kafka / AMQP / inproc
//
// # Choisir le relais
//
//	MESSAGING_DRIVER=inproc     tout dans le même processus (défaut)
//	MESSAGING_DRIVER=kafka      MESSAGING_BROKERS=host:9092
//	MESSAGING_DRIVER=rabbitmq   MESSAGING_AMQP_URL=amqp://…
//	MESSAGING_DRIVER=noop       aucune publication (tests, migrations)
package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
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

// ErrUnknownDriver signale un MESSAGING_DRIVER non reconnu.
//
// Deny par défaut : un pilote inconnu refuse le démarrage plutôt que de se
// rabattre silencieusement sur « aucune publication ».
var ErrUnknownDriver = errors.New("relais de messagerie inconnu")

// Closer libère les ressources du transport.
type Closer = func() error

// New construit le publieur et le consommateur correspondant à la configuration.
//
// C'est le SEUL point du dépôt qui choisit un broker. Ajouter un transport se
// fait ici et nulle part ailleurs.
func New(cfg config.Messaging, logger *slog.Logger) (Publisher, Consumer, Closer, error) {
	switch Driver(cfg.Driver) {
	case DriverInproc:
		bus := NewInproc(logger)
		return bus.Publish, bus, func() error { return nil }, nil
	case DriverNoop:
		return noopPublisher(logger), noopConsumer{}, func() error { return nil }, nil
	case DriverKafka:
		return newKafka(cfg, logger)
	case DriverRabbitMQ:
		return newRabbitMQ(cfg, logger)
	default:
		return nil, nil, nil, fmt.Errorf("%w: %q", ErrUnknownDriver, cfg.Driver)
	}
}

// Inproc est un bus en mémoire : les consommateurs tournent dans le même
// processus que le producteur.
//
// Ce n'est pas un bouchon. C'est le mode NORMAL d'un monolithe modulaire : la
// durabilité est déjà assurée par l'outbox, le bus n'a donc pas à l'être. On ne
// paie un broker que le jour où les modules sont déployés séparément.
type Inproc struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   *slog.Logger
}

// NewInproc construit le bus en mémoire.
func NewInproc(logger *slog.Logger) *Inproc {
	return &Inproc{handlers: make(map[string][]Handler), logger: logger}
}

// Subscribe enregistre un consommateur.
func (b *Inproc) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish remet l'enveloppe aux consommateurs, de façon synchrone.
//
// Synchrone volontairement : l'appelant est le worker, qui ne marquera le
// message traité qu'après succès. Publier en asynchrone ici perdrait la
// garantie de livraison que l'outbox vient d'apporter.
func (b *Inproc) Publish(ctx context.Context, env Envelope) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[env.Type]...)
	b.mu.RUnlock()

	if len(handlers) == 0 {
		b.logger.WarnContext(ctx, "événement sans consommateur",
			slog.String("event_type", env.Type))
		return nil
	}
	var failures []error
	for _, handler := range handlers {
		if err := handler(ctx, env); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("consommation de %s: %w", env.Type, errors.Join(failures...))
	}
	return nil
}

// Run ne fait rien : les consommateurs sont appelés par Publish.
func (b *Inproc) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func noopPublisher(logger *slog.Logger) Publisher {
	return func(ctx context.Context, env Envelope) error {
		logger.DebugContext(ctx, "publication ignorée (relais noop)",
			slog.String("event_type", env.Type))
		return nil
	}
}

type noopConsumer struct{}

func (noopConsumer) Subscribe(string, Handler) {}

func (noopConsumer) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// WithRetry enveloppe un publieur d'un réessai borné.
//
// Utile pour les transports réseau : une coupure d'une seconde ne doit pas
// consommer une tentative de l'outbox, dont le recul est bien plus long.
func WithRetry(publisher Publisher, attempts int, wait time.Duration) Publisher {
	return func(ctx context.Context, env Envelope) error {
		var last error
		for i := range attempts {
			if i > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait * time.Duration(1<<(i-1))):
				}
			}
			last = publisher(ctx, env)
			if last == nil {
				return nil
			}
		}
		return fmt.Errorf("publication échouée après %d tentatives: %w", attempts, last)
	}
}
