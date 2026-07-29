package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

// Inproc is an in-memory bus: the consumers run in the same process as the
// producer.
//
// This is not a stub. It is the NORMAL mode of a modular monolith: durability
// is already ensured by the outbox, so the bus does not have to provide it. One
// only pays for a broker the day the modules are deployed separately.
type Inproc struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	logger   *slog.Logger
}

// NewInproc builds the in-memory bus.
func NewInproc(logger *slog.Logger) *Inproc {
	return &Inproc{handlers: make(map[string][]Handler), logger: logger}
}

// Subscribe registers a consumer.
func (b *Inproc) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish hands the envelope over to the consumers, synchronously.
//
// Synchronous deliberately: the caller is the worker, which will only mark the
// message as handled after success. Publishing asynchronously here would lose
// the delivery guarantee the outbox has just brought.
func (b *Inproc) Publish(ctx context.Context, env Envelope) error {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[env.Type]...)
	b.mu.RUnlock()

	if len(handlers) == 0 {
		b.logger.WarnContext(ctx, "event without consumer",
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
		return fmt.Errorf("consumption of %s: %w", env.Type, errors.Join(failures...))
	}
	return nil
}

// Run does nothing: the consumers are called by Publish.
func (b *Inproc) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
