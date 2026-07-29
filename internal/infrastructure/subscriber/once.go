package subscriber

import (
	"context"
	"errors"
	"fmt"

	idempotencydomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	idempotencyports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// Guard carries the three idempotency ports the decorator needs.
//
// A struct rather than three parameters: `Once(handler, reserve, complete,
// release)` makes four arguments, and above all leaves the caller free to swap
// two of them — `complete` and `release` have the same signature, and the swap
// would release the key on every success while locking every failure. The
// compiler would see nothing.
type Guard struct {
	Reserve  idempotencyports.Reserve
	Complete idempotencyports.Complete
	Release  idempotencyports.Release
}

// Once makes the effect take place only ONCE, even if the envelope comes back.
//
// # The protocol, and why each branch is there
//
//	reserve → already replayed? acknowledge without doing anything
//	        → in flight?        RETURN AN ERROR so the transport replays
//	        → otherwise         run, then complete or release
//
// **A replay acknowledges without running.** That is the whole object: the
// transports here are "at least once", so the same envelope arrives twice as
// soon as an acknowledgement is lost. Without this guard, a welcome email
// leaves twice — and on the day the consumer debits a card, it will be the
// debit.
//
// **An "in flight" returns an ERROR, deliberately.** Another replica is already
// handling that envelope and may still fail. Acknowledging it here would lose
// the event if the other one gives up. Replaying later costs only one attempt;
// acknowledging wrongly costs the effect.
//
// **`Release` is in a `defer` on the failure path.** Forgetting it would make
// the envelope unhandleable until the key expires — that is twenty-four hours
// during which a replay is refused without anything explaining why.
func Once(guard Guard, handler messaging.Handler) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		if env.ID == "" {
			return fmt.Errorf("%w: type=%q", ErrMissingID, env.Type)
		}

		key := idempotencydomain.Key(env.ID)
		reservation, err := guard.Reserve(ctx, idempotencydomain.Request{
			Key: key,
			// The fingerprint is the TYPE and not the payload: two envelopes
			// with the same identifier carry the same fact. Fingerprinting the
			// payload would make a replay fail in conflict when the producer
			// had serialised it differently — a JSON key order is enough.
			Fingerprint: env.Type,
		})
		switch {
		case errors.Is(err, idempotencydomain.ErrInFlight):
			return fmt.Errorf("envelope %s already being handled: %w", env.ID, err)
		case err != nil:
			return fmt.Errorf("reservation of %s: %w", env.ID, err)
		case reservation.Replayed:
			// Already handled. We acknowledge without re-running: it is the only
			// branch that returns `nil` without having done anything, and it is
			// the reason this package exists.
			return nil
		}

		if err := handler(ctx, env); err != nil {
			// Release BEFORE returning, so that a replay is possible straight
			// away. The release error is joined rather than overwritten: a key
			// left locked is a distinct incident, which is hard to diagnose when
			// its message has disappeared.
			return errors.Join(
				fmt.Errorf("handling of %s: %w", env.ID, err),
				release(ctx, guard, key),
			)
		}

		if err := guard.Complete(ctx, key, nil); err != nil {
			// The effect HAS TAKEN PLACE. Returning the error will make the
			// envelope be replayed, and the replay will NOT be recognised since
			// the key is not completed: the effect will therefore take place
			// twice. It is the only case where this package does not keep its
			// promise, and it is named rather than kept quiet — same shape as
			// the "published but not marked" of the outbox dispatcher.
			return fmt.Errorf("envelope %s handled but NOT marked: %w", env.ID, err)
		}
		return nil
	}
}

// release releases the key and wraps its error.
func release(ctx context.Context, guard Guard, key idempotencydomain.Key) error {
	if err := guard.Release(ctx, key); err != nil {
		return fmt.Errorf("release of key %s: %w", key, err)
	}
	return nil
}
