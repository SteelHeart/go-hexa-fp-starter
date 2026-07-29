// Package ports declares the contracts of object storage.
//
// This package contains ONLY type declarations.
package ports

import (
	"context"
	"io"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// Put writes an object and returns its location.
//
// Contract: the key is ALWAYS derived by domain.SafeKey, never built by the
// driver. A refused name returns domain.ErrUnsafeName, whichever driver is in
// use.
//
// The content is consumed as a stream: a driver does not load the whole object
// into memory, otherwise a two-gigabyte upload would kill the process.
type Put = func(ctx context.Context, obj domain.Object) (domain.Located, error)

// Get reads a stored object.
//
// The caller MUST close the returned stream. A key outside the bounds of the
// store returns domain.ErrUnsafeName — the key comes from a URL, hence from a
// stranger.
type Get = func(ctx context.Context, key domain.Key) (io.ReadCloser, error)

// Delete removes an object.
//
// Contract: deleting an absent object is NOT an error. The caller wants the
// object to be gone, and it is gone. Making an idempotent deletion fail would
// force every caller to distinguish two equivalent cases.
type Delete = func(ctx context.Context, key domain.Key) error
