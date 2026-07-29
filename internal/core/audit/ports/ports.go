// Package ports declares the contract of the audit log.
//
// This package contains only type declarations.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// Record records an audit fact.
//
// Contract: the write goes through the querier of the context, hence INSIDE
// the business transaction if there is one. A rolled back fact leaves no lying
// audit trace — that is what distinguishes an audit log from a log.
//
// Error contract: an incomplete entry is refused (see Entry.IsComplete). Any
// other error is technical.
type Record = func(ctx context.Context, entry domain.Entry) error
