package messaging

import (
	"context"
	"log/slog"
)

// noopPublisher publishes nothing, and SAYS SO.
//
// The Debug trace is what tells "relay deliberately disabled" apart from "relay
// broken": a silent transport that logs nothing looks exactly like a transport
// that has broken down.
func noopPublisher(logger *slog.Logger) Publisher {
	return func(ctx context.Context, env Envelope) error {
		logger.DebugContext(ctx, "publication discarded (noop relay)",
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
