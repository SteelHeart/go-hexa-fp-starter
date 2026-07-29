// Package application composes the notification use cases.
//
// It does not log and does not read the clock: it reports through its return
// values. That is what keeps it testable without parsing logs.
//
// # File map
//
//	send.go   convey an already rendered message
package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/ports"
)

// Deps carries the ports the use cases need.
//
// All are function types: in a test, each is a three-line closure, and no
// mocking library is necessary — hence none is allowed
// (rules/dependances.md).
type Deps struct {
	Deliver ports.Deliver
}

// NewSend composes the conveyance of a message.
//
// # Two useful lines, and that is the point
//
// Validate, deliver. There is nowhere to slip in a template, a translation, or a
// retry policy — all three are application decisions, not starter ones, and
// burying them here would impose them on everyone.
//
// The revalidation is not paranoia: `domain.Message` has exported fields, so a
// caller can build one without going through `NewMessage`. The use case is the
// last place where one can still refuse before an empty address reaches a
// provider — where it would become a billed rejection logged at a third party.
func NewSend(deps Deps) ports.Send {
	return func(ctx context.Context, message domain.Message) error {
		validated, err := domain.NewMessage(
			message.Channel, message.To, message.Subject, message.Body)
		if err != nil {
			return fmt.Errorf("message: %w", err)
		}
		if err := deps.Deliver(ctx, validated); err != nil {
			return fmt.Errorf("delivery to the provider: %w", err)
		}
		return nil
	}
}
