package messaging

import (
	"context"
	"log/slog"
)

// noopPublisher ne publie rien, et le DIT.
//
// La trace en Debug est ce qui distingue « relais volontairement désactivé » de
// « relais cassé » : un transport muet qui ne journalise rien ressemble
// exactement à un transport en panne.
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
