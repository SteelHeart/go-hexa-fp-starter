// Package ports declares the notification contracts.
//
// This package contains ONLY type declarations: no struct, no function, no
// interface. A port is a function type — the smallest possible interface, hence
// nothing to segregate (ADR 003).
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
)

// ─── Primary port: what the outside world may ask for ────────────────────────

// Send conveys a message.
//
// # What this port does NOT promise
//
// Neither reception, nor reading, nor even final delivery. It promises that the
// message has been ACCEPTED by the provider. An accepted email may still be
// rejected by the recipient server a minute later, and no synchronous code can
// know it.
//
// Promising more would lead to callers treating success as proof of reception —
// and that belief is paid for at the first customer support ticket claiming
// nothing was received.
//
// Errors: `domain.ErrIncomplete` for a malformed message,
// `domain.ErrUnknownChannel` for an unserved channel, `domain.ErrUndeliverable`
// when the provider refuses. The distinction decides the retry — one replays an
// outage, never an invalid address.
type Send = func(ctx context.Context, message domain.Message) error

// ─── Secondary port: what the core needs from the world ──────────────────────

// Deliver hands the message over to the provider.
//
// Receives an ALREADY VALIDATED message: the driver does not have to revalidate,
// and above all must not. Two validations for one rule is one too many — they
// will diverge, and it is the wrong one that will decide.
//
// Error contract: `domain.ErrUndeliverable`, wrapped with the cause.
type Deliver = func(ctx context.Context, message domain.Message) error
