package messaging

import (
	"context"
	"fmt"
	"time"
)

// Réessai appliqué aux deux relais réseau.
//
// Il absorbe la coupure d'une seconde ; il ne remplace PAS le recul de l'outbox,
// qui se compte en minutes et survit à un redémarrage du processus.
const (
	publishAttempts = 3
	publishBackoff  = 200 * time.Millisecond
)

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
