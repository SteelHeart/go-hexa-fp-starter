package tests

import (
	"context"
	"errors"
	"testing"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
)

// TestPublicationFailureReachesTheDispatcher: a publication failure goes back
// up intact.
//
// The exponential backoff and the abandonment after N attempts are decided by a
// PURE and tested policy, in outbox/application. This bridge therefore has
// nothing to decide.
//
// The two ways of getting it wrong here are serious and opposite:
//
//   - Swallowing the error would mark the message as HANDLED although nothing
//     has left. The event would be definitively lost, without any trace — the
//     very defect the outbox exists precisely to make impossible.
//   - Catching it to decide by itself would duplicate the backoff policy, and
//     the two would diverge on the first adjustment.
func TestPublicationFailureReachesTheDispatcher(t *testing.T) {
	t.Parallel()

	unreachable := errors.New("broker unreachable")
	handle := relay.FromOutbox(func(context.Context, messaging.Envelope) error {
		return unreachable
	})

	err := handle(context.Background(), outboxdomain.Message{
		ID:   "019f9b46-3aec-735a-977d-129192ef130f",
		Type: "user.registered.v1",
	})

	if err == nil {
		t.Fatal("a publication failure must NEVER be swallowed: the message would be marked as handled")
	}
	if !errors.Is(err, unreachable) {
		t.Errorf("error = %v, the cause must reach the dispatcher intact", err)
	}
}
