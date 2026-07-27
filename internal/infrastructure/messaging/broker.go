package messaging

import (
	"fmt"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Broker est un relais monté : ses trois faces vont ensemble.
//
// Les rendre séparément avait fait quatre valeurs de retour, et surtout laissait
// l'appelant libre d'en oublier une — typiquement Close, dont l'oubli ne se voit
// qu'au moment où les connexions au broker fuient en production.
type Broker struct {
	// Publish remet une enveloppe au transport.
	Publish Publisher
	// Consume s'abonne et boucle jusqu'à annulation.
	Consume Consumer
	// Close libère les ressources. Jamais nil, même pour un relais qui n'en
	// détient aucune : l'appelant ne doit pas avoir à tester.
	Close Closer
}

// noClose est le libérateur d'un relais sans ressource externe.
func noClose() error { return nil }

// New construit le relais correspondant à la configuration.
//
// C'est le SEUL point du dépôt qui choisit un broker. Ajouter un transport se
// fait ici et nulle part ailleurs.
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
