// Package ports declares the contracts of dynamic configuration.
//
// This package contains ONLY type declarations.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// IsEnabled answers "is this flag active?".
//
// # Why this port does NOT return an error
//
// This is the only exception to the rule "a core module returns error", and it
// is a deliberate one. The contract of the port IS the fallback: unknown flag,
// unreachable store, unreadable value — the answer is `false` in all three
// cases.
//
// Returning `(bool, error)` would force every call — there is one per hidden
// branch of code — to handle an error whose only correct response is already
// known. In practice, those sites would write `enabled, _ := flag(ctx, key)`,
// and the fallback would become implicit instead of being guaranteed by the
// port.
//
// The trade-off is real and must be known: an outage of the store silently
// SWITCHES OFF the hidden features. A driver that fails must therefore log it
// itself — that is what the contract requires of it in exchange.
type IsEnabled = func(ctx context.Context, key domain.FlagKey) bool

// GetSetting reads a business setting. An absence is not an error: it is the
// nominal case of a setting that has never been set.
type GetSetting = func(ctx context.Context, key domain.SettingKey) domain.Setting

// Set writes a dynamic value.
//
// Returns `domain.ErrReadOnly` on a store that does not accept writes. A driver
// that accepted the write without persisting it would be worse: the caller
// would believe it had changed something.
type Set = func(ctx context.Context, change domain.Change) error

// Invalidate purges the local cache.
//
// With neither context nor error: purging an in-memory cache can neither fail,
// nor wait. To be called after a write made by ANOTHER replica.
type Invalidate = func()
