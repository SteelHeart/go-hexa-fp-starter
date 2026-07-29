package messaging

import (
	"context"
	"fmt"
	"time"
)

// Retry applied to both network relays.
//
// It absorbs a one-second outage; it does NOT replace the backoff of the
// outbox, which is counted in minutes and survives a restart of the process.
const (
	publishAttempts = 3
	publishBackoff  = 200 * time.Millisecond
)

// WithRetry wraps a publisher in a bounded retry.
//
// Useful for network transports: a one-second outage must not consume an
// attempt of the outbox, whose backoff is far longer.
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
		return fmt.Errorf("publication failed after %d attempts: %w", attempts, last)
	}
}
