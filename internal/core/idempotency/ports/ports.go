// Package ports declares the contracts of idempotency.
//
// This package contains ONLY type declarations: no struct, no function, no
// interface.
//
// # Why `error` and not `Result[T, domain.Error]`
//
// A CORE module is technical: idempotency has no business error taxonomy to
// expose. The boundary is sharp and verifiable — `internal/core/**` uses
// `error`, `internal/modules/**` uses `Result`.
//
// # The protocol, in three calls
//
//	res, err := reserve(ctx, domain.Request{Key: k, Fingerprint: fp})
//	switch {
//	case errors.Is(err, domain.ErrInFlight): // 409: retry, do NOT execute
//	case errors.Is(err, domain.ErrConflict): // 422: the client is at fault
//	case err != nil:                        // technical failure
//	case res.Replayed:                      // return res.Response as it is
//	default:                                // execute, then complete() or release()
//	}
//
// Forgetting `release()` on failure makes the operation impossible until the key
// expires: the remedy would be worse than the disease. `defer` is the right
// reflex.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// Reserve attempts to obtain exclusivity for a key.
//
// # Conformance contract — every driver must return these five outcomes
//
//  1. Incomplete request → `domain.ErrIncomplete`. Never an empty key accepted.
//  2. Free key or expired reservation → `Reservation{}` and the caller executes.
//  3. Known key, different fingerprint → `domain.ErrConflict`.
//  4. Known key, completed operation → `Reservation{Replayed: true, Response: …}`.
//  5. Known key, operation in flight → `domain.ErrInFlight`.
//
// Outcome 5 demands atomicity: two concurrent calls on a free key can NEVER both
// obtain outcome 2. A driver unable to guarantee it is not conformant — this is
// the module's only promise.
//
// When in doubt, a driver refuses (`ErrInFlight`) rather than allows: making a
// client retry is benign, executing twice is not.
type Reserve = func(ctx context.Context, req domain.Request) (domain.Reservation, error)

// Complete memorises the response for later replays.
//
// Returns `domain.ErrNotReserved` if the reservation no longer exists. The
// business operation itself succeeded: the caller logs and carries on.
type Complete = func(ctx context.Context, key domain.Key, response []byte) error

// Release frees a key after a failure, so that the caller can retry.
//
// NEVER frees a completed key: that would reopen the door to the replay the
// module exists to close.
type Release = func(ctx context.Context, key domain.Key) error

// Purge deletes expired keys and returns their number.
//
// To be called periodically by the scheduler. A driver whose store expires on
// its own returns 0 without error.
type Purge = func(ctx context.Context) (int64, error)
