// Package ports declares the contracts of the outbox.
//
// This package contains ONLY type declarations: no struct, no function, no
// interface (checked by arch-go.yml).
//
// # Why `error` and not `Result[T, domain.Error]`
//
// A CORE module is technical: the outbox has no business error taxonomy to
// expose. `rules/programmation-fonctionnelle.md` §4 reserves `Result` for the
// business core and admits bare `error` in the technical part. A BUSINESS
// module, on the other hand, always returns `Result[T, domain.Error]`.
//
// This boundary is sharp and verifiable: `internal/core/**` uses `error`,
// `internal/modules/**` uses `Result`.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// Enqueue persists an intent to publish.
//
// To be called INSIDE the business transaction: that is the whole point of the
// pattern. The `postgres` driver reads the querier from the context, so
// atomicity is obtained without the caller having to bother.
//
// Error contract: every error is technical. The `memory` driver can only fail
// on a cancelled context.
type Enqueue = func(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error)

// Claim claims a batch of due messages and locks them.
//
// Contract: two concurrent calls NEVER return the same message. The `postgres`
// driver obtains this through `FOR UPDATE SKIP LOCKED`, the `memory` driver
// through a local lock. A driver that cannot guarantee it is not conformant.
type Claim = func(ctx context.Context, limit int) ([]domain.Message, error)

// MarkDone marks a message as published successfully.
type MarkDone = func(ctx context.Context, id domain.MessageID) error

// MarkFailed records a failure and schedules the next attempt.
//
// The backoff computation is NOT here: it is in domain.NextAttempt, hence pure
// and testable. The driver only writes.
type MarkFailed = func(ctx context.Context, attempt domain.FailedAttempt) error

// PendingCount counts the pending messages.
//
// This is the most important metric of the system: it grows when the worker is
// dead, and it is the only visible symptom of a broken asynchronous chain.
type PendingCount = func(ctx context.Context) (int64, error)

// Handler processes a message.
//
// It MUST be idempotent: dispatching is « at least once », so every message
// will be replayed at least once in the lifetime of the system.
type Handler = func(ctx context.Context, msg domain.Message) error

// Report reports on the processing of a message.
//
// Orchestration does not log: it reports. That is what keeps it pure —
// `rules/README.md` forbids any logger in `application/` — and what allows a
// test to check a dispatching policy by reading values.
//
// Returns nothing: a report that failed must never make the dispatching it
// reports on fail.
type Report = func(ctx context.Context, outcome domain.Outcome)

// Now returns the current instant.
//
// Orchestration does not read the system clock: it receives this port. Without
// it, the backoff policy would be untestable without actually waiting.
type Now = func() time.Time
