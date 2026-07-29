package messaging

import (
	"fmt"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Broker is a mounted relay: its three faces go together.
//
// Returning them separately made four return values, and above all left the
// caller free to forget one — typically Close, whose omission only shows up the
// moment connections to the broker leak in production.
type Broker struct {
	// Publish hands an envelope over to the transport.
	Publish Publisher
	// Consume subscribes and loops until cancellation.
	Consume Consumer
	// Close releases the resources. Never nil, even for a relay that holds
	// none: the caller must not have to test.
	Close Closer
}

// noClose is the releaser of a relay without external resources.
func noClose() error { return nil }

// New builds the relay matching the configuration.
//
// This is the ONLY point of the repository that chooses a broker. Adding a
// transport happens here and nowhere else.
func New(cfg config.Messaging, logger *slog.Logger) (Broker, error) {
	switch Driver(cfg.Driver) {
	case DriverInproc:
		bus := NewInproc(logger)
		return Broker{Publish: bus.Publish, Consume: bus, Close: noClose}, nil
	case DriverNoop:
		return Broker{Publish: noopPublisher(logger), Consume: noopConsumer{}, Close: noClose}, nil
	case DriverKafka:
		return newKafka(cfg, logger)
	case DriverRabbitMQ:
		return newRabbitMQ(cfg, logger)
	default:
		return Broker{}, fmt.Errorf("%w: %q", ErrUnknownDriver, cfg.Driver)
	}
}
