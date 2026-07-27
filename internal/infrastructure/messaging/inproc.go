package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

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
