// Package domain carries the vocabulary of idempotency, without dependencies.
//
// # Why this module exists
//
// As soon as a mobile frontend is served, the network is unstable: the client
// resends its request without knowing whether the first one went through.
// Without an idempotency key, a replayed POST creates two resources — and the
// duplicate is discovered by the user, not by the service.
package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Key is the key supplied by the caller. Two requests carrying the same key
// designate the SAME intent.
type Key string

// String returns the key in raw form, for the drivers that write it.
func (k Key) String() string { return string(k) }

// Status is the state of a reservation. Declared here and not in a driver: all
// drivers must write the same vocabulary, otherwise changing driver would make
// existing memorisations unreadable.
type Status string

const (
	// StatusInFlight marks an operation started and not resolved.
	StatusInFlight Status = "in_flight"
	// StatusDone marks a completed operation whose response is memorised.
	StatusDone Status = "done"
)

// ErrConflict reports a key reused with a different fingerprint.
//
// This is a defect of the client: the same key must designate the same request,
// otherwise the guarantee no longer means anything. We refuse rather than guess
// which of the two requests is the right one.
var ErrConflict = errors.New("idempotency key reused with a different request")

// ErrInFlight reports that an identical request is already in flight.
//
// The caller must retry later, never execute. This is the refusal that makes the
// guarantee true under concurrency.
var ErrInFlight = errors.New("identical request already in flight")

// ErrIncomplete refuses a request without a key or without a fingerprint.
//
// An empty key would be shared by ALL callers: the first replay from anyone
// would mask the operation of anyone else. A driver that accepted the empty key
// would turn a protection into a data leak.
var ErrIncomplete = errors.New("incomplete idempotency request")

// ErrNotReserved reports a memorisation without a live reservation.
//
// Happens when the reservation expired during the operation. The caller has
// already executed: it must log it, never roll back. This error says "the
// memorisation failed", not "the operation failed".
var ErrNotReserved = errors.New("no live reservation for this key")

// Request is what a key commits to.
type Request struct {
	Key Key
	// Fingerprint is the fingerprint of the payload, computed by Fingerprint.
	Fingerprint string
}

// IsComplete reports whether the request carries the usable minimum.
func (r Request) IsComplete() bool {
	return r.Key != "" && r.Fingerprint != ""
}

// Reservation is the outcome of an obtained reservation.
//
// # Why a two-field struct and not fp.Option
//
// The ports of a core module depend on nothing but their domain
// (arch-go.yml §4e): they cannot name `fp.Option`. The gain is real —
// `Replayed` reads at the call site, where an empty `Option` would be confused
// with "memorised but empty response".
type Reservation struct {
	// Replayed set to true means: DO NOT execute, return Response as it is.
	Replayed bool
	// Response is the memorised response of the first call. Nil if Replayed is
	// false.
	Response []byte
}

// Fingerprint computes the fingerprint of a payload.
//
// Deterministic: encoding/json orders the keys of a map, so two calls on the
// same value return the same fingerprint. A change in the shape of the payload
// changes the fingerprint, which is exactly the point — the key must not cover
// two different requests.
func Fingerprint(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		// A non-serialisable payload is a programming defect, not a runtime
		// error. We return a fingerprint that does not collide rather than
		// propagate an error on a purely defensive path.
		raw = []byte(fmt.Sprintf("%#v", payload))
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
