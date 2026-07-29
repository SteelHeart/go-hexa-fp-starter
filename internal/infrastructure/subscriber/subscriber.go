// Package subscriber plugs an event consumer onto the guarantees of the core.
//
// # Why this package exists, and why HERE
//
// It is the exact counterpart of `relay`: the latter links the dispatcher to
// the transport, this one links the transport to the modules that react.
// Neither of the two can live in a core module — `idempotency` must import no
// infrastructure, on pain of no longer being extractable into an independent Go
// module (ADR 012) — nor in `cmd/`, because code inside `main` is only half
// testable.
//
// # What this package guards, and what nobody notices when it works
//
// Three things every consumer must do and everybody forgets:
//
//  1. **Do not replay an effect.** Every transport here is "at least once". A
//     welcome email sent twice is the visible symptom; a replayed debit is the
//     costly one.
//  2. **Restore the trace.** Without it, the asynchronous half of a request
//     appears as an orphan trace, and nobody ever links the sending failure to
//     the registration that caused it.
//  3. **Do not confuse "already done" with "failure".** The first acknowledges,
//     the second replays.
//
// # File map
//
//	subscriber.go   the map of the package and the errors
//	once.go         the effect takes place only ONCE, even on a replay
//	trace.go        the trace context taken back from the envelope
package subscriber

import "errors"

// ErrMissingID refuses an envelope without an identifier.
//
// The identifier IS the idempotency key: without it, no replay can be
// recognised, and the decorator would let every copy through believing it was
// doing the right thing. A noisy refusal is worth more than a silently absent
// guarantee.
var ErrMissingID = errors.New("envelope without an identifier: idempotency impossible")
