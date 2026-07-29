// Package log implements notification towards the application log.
//
// # Why this driver exists
//
// It is not a test stub: it is the DEFAULT driver, and it is what makes the
// starter's promise true — `hexa new` then `go run` starts the complete chain,
// **registration → outbox → relay → notification**, without an SMTP server,
// without a provider account, without Docker.
//
// # GUARANTEES
//
//   - **The body is NOT logged by default.** A notification body regularly
//     carries a secret — confirmation link, reset token, one-time code. These
//     are bearer credentials: whoever reads them can use them.
//   - **The address is masked**, always, including when the body is written. An
//     address is personal data (rules/securite.md §5), and the application log
//     is what one exports the most willingly to a third party.
//   - **No error is invented.** Writing to a log does not fail here, so this
//     driver always returns nil.
//
// # NON-GUARANTEES
//
//   - **Nobody receives anything.** This is an OBSERVATION driver: the message
//     is written, not sent. Using it anywhere but in development would make one
//     believe the emails are going out — and the defect would only show at the
//     first customer claiming they received nothing.
//   - **No retry, no queue.** It does not need them; an SMTP driver will have
//     them, and that will be its own business.
package log

import (
	"context"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/ports"
)

// New builds the driver WITHOUT logging the body.
//
// This is the default constructor, and the safe one. Two named constructors
// rather than a boolean parameter: `New(logger, false)` does not read at the
// call site, and `New(logger, true)` slips through a review unnoticed. The
// repository has already split `SecurityHeaders(secure bool)` for this reason.
func New(logger *slog.Logger) ports.Deliver {
	return func(ctx context.Context, message domain.Message) error {
		logger.InfoContext(ctx, "notification", fields(message)...)
		return nil
	}
}

// NewIncludingBody builds the driver logging the body.
//
// # To be asked for in development only, and the name says so
//
// Without the body, one cannot retrieve the confirmation link of a test account
// — and development becomes painful enough that someone eventually adds a
// misplaced `fmt.Println`, which will never be removed. The trade-off is
// therefore real in both directions; what matters is that the dangerous position
// must be ASKED FOR, instead of being the default.
func NewIncludingBody(logger *slog.Logger) ports.Deliver {
	return func(ctx context.Context, message domain.Message) error {
		// The warning travels in the SAME record as the body: whoever reads the
		// second reads the first. In two separate lines, the first gets lost in
		// the sorting.
		withBody := append(fields(message),
			slog.String("body", message.Body),
			slog.String("warning", "body logged — development only"),
		)
		logger.InfoContext(ctx, "notification", withBody...)
		return nil
	}
}

// fields returns what is logged in ALL cases.
//
// The address is masked there and the body reduced to its size — useful for
// diagnosis ("are the messages going out empty?") without revealing anything of
// their content.
func fields(message domain.Message) []any {
	return []any{
		slog.String("channel", string(message.Channel)),
		slog.String("to", message.To.Masked()),
		slog.String("subject", message.Subject),
		slog.Int("body_bytes", len(message.Body)),
	}
}
