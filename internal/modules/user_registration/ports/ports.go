// Package ports declares the contracts of the feature.
//
// This package contains ONLY type declarations: no struct, no function, no
// interface (enforced by arch-go.yml). A port is a function type — the smallest
// possible interface, hence nothing to segregate (documentation/adr/003).
//
// Every secondary port carries its ERROR CONTRACT in a comment: that is the
// operational form of substitutability, and it is checked by the conformance
// suite shared between implementations.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// ─── Primary ports: what the outside world may ask for ───────────────────────

// RegisterUser registers a new user.
//
// Every surface calls THIS function: HTTP, CLI, seeders. Adding a surface does
// not change its signature (documentation/adr/005).
//
// Errors: CodeInvalidEmail · CodeWeakPassword · CodeEmailAlreadyExists ·
// CodeUnavailable · CodeInternal.
type RegisterUser = func(
	ctx context.Context,
	cmd domain.RegistrationCommand,
) result.Result[domain.User, domain.Error]

// CheckEmailAvailability reports whether an email address is still free.
//
// A READ port: it is the one that carries the cache decorator, a cache on a
// write making no sense at all.
//
// Errors: CodeInvalidEmail · CodeUnavailable · CodeInternal.
type CheckEmailAvailability = func(
	ctx context.Context,
	rawEmail string,
) result.Result[bool, domain.Error]

// ─── Secondary ports: what the core needs from the world ─────────────────────

// SaveUser persists a newly registered user.
//
// Error contract — every implementation MUST honour:
//   - CodeEmailAlreadyExists if the address uniqueness constraint is violated
//   - CodeUnavailable        if the storage is unreachable or the query cancelled
//   - CodeInternal           for everything else, with the cause attached
//
// No driver error must travel up as it is.
type SaveUser = func(
	ctx context.Context,
	user domain.User,
) result.Result[domain.User, domain.Error]

// EmailIsTaken reports whether an address is already registered.
//
// Error contract: CodeUnavailable · CodeInternal. A missing row is NOT an error
// — it amounts to `false`.
type EmailIsTaken = func(
	ctx context.Context,
	email domain.Email,
) result.Result[bool, domain.Error]

// FindUserByEmail looks a user up by email address.
//
// Error contract: CodeUnavailable · CodeInternal. An absence returns `None`,
// never an error: the Option makes absence explicit in the type.
type FindUserByEmail = func(
	ctx context.Context,
	email domain.Email,
) result.Result[fp.Option[domain.User], domain.Error]

// HashPassword produces the digest of a password.
//
// Error contract: CodeInternal only. A hashing failure is a technical defect,
// never a user error.
type HashPassword = func(
	password domain.RawPassword,
) result.Result[domain.PasswordHash, domain.Error]

// PublishEvent records an event to be published.
//
// The implementation writes into the outbox, WITHIN the current transaction. The
// core knows no broker (documentation/adr/006).
//
// Error contract: CodeUnavailable · CodeInternal.
type PublishEvent = func(
	ctx context.Context,
	eventType string,
	aggregateID string,
	payload any,
) result.Result[domain.Ack, domain.Error]

// SendWelcomeEmail sends the welcome email.
//
// Called by the event consumer, never by the registration use case: sending must
// not make the registration fail.
//
// Error contract: CodeUnavailable · CodeInternal.
type SendWelcomeEmail = func(
	ctx context.Context,
	user domain.User,
) result.Result[domain.Ack, domain.Error]

// ─── Ports for pure effects: the clock and randomness ────────────────────────

// Now returns the current instant.
//
// It is a port PRECISELY so that tests are deterministic: a test that reads the
// real clock is a test that will fail one day, for no reason.
type Now = func() time.Time

// GenerateID produces a user identifier.
type GenerateID = func() domain.UserID

// ─── Unit of work ────────────────────────────────────────────────────────────

// RunInTx runs a function inside a transaction.
//
// The rollback is triggered by a Result in Err: a refused registration therefore
// leaves no event in the outbox.
type RunInTx = func(
	ctx context.Context,
	fn func(context.Context) result.Result[domain.User, domain.Error],
) result.Result[domain.User, domain.Error]
