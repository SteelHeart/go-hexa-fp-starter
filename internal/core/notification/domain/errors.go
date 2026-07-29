// Package domain carries the notification vocabulary, with no dependency.
//
// # Why this module exists
//
// `user.registered.v1` has been published since the first vertical slice, and
// **nobody subscribes to it**. The chain stops at the relay: the event is
// durable, transported, and without effect. A module that produces an event
// nothing consumes is a module whose asynchronous half has never run — and this
// repository has already measured what never-executed code costs.
//
// # What this module does NOT know
//
// It knows nothing of `user_registration`, of registration, or of the word
// "welcome". It sends a message to a recipient over a channel. It is the
// composition root that links a business event to a message — and that
// ignorance is what makes the module reusable by `billing` tomorrow without
// touching it.
//
// # This package is PURE
//
// No clock, no I/O, no template. Rendering content belongs to the caller: a core
// module that embedded a template engine would impose its syntax on every
// application of the starter.
package domain

import "errors"

// Sentinel errors of the module.
//
// `internal/core/**` returns `error` and not `Result[T, domain.Error]`: the
// boundary is clean and verifiable, and a core module is technical.
var (
	// ErrIncomplete refuses a malformed message BEFORE any send.
	//
	// The refusal is upstream for two reasons: it costs no network call, and it
	// does not let an empty address reach a provider, where it would become a
	// billed rejection logged at a third party.
	ErrIncomplete = errors.New("incomplete message")

	// ErrUnknownChannel refuses a channel this module cannot serve.
	//
	// Deny by default: an unknown channel never resolves to "the closest one".
	// Falling back to email because SMS is not shipped would send a verification
	// code to the wrong address.
	ErrUnknownChannel = errors.New("unknown notification channel")

	// ErrUndeliverable signals a provider failure.
	//
	// Distinct from `ErrIncomplete`: the message was valid, it is the send that
	// failed. The difference decides the retry — one replays an outage, never an
	// invalid address.
	ErrUndeliverable = errors.New("notification not delivered")
)
