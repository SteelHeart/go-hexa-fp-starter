package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestUnmarkableMessageIsReportedAsDuplicateRisk covers the most serious and
// least obvious case of dispatching.
//
// The message has been PUBLISHED, but the database went down before it could be
// recorded. It therefore stays `pending` and will go out again on the next
// round: the consumer will receive a duplicate.
//
// This is not a defect that can be removed — publishing and recording are two
// systems, and no transaction covers them together. That is exactly why
// delivery is « at least once » and why `ports.Handler` imposes idempotency on
// the consumer. What IS demandable is that the risk be reported distinctly
// rather than confused with a publication failure.
func TestUnmarkableMessageIsReportedAsDuplicateRisk(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	var published bool
	handle := func(context.Context, domain.Message) error { published = true; return nil }

	failingMarkDone := func(context.Context, domain.MessageID) error {
		return errors.New("database unreachable")
	}

	dispatcher := newDispatcher(t, application.Ports{
		Claim:      claimOnce(pending("m-1", 0)),
		MarkDone:   failingMarkDone,
		MarkFailed: observed.markFailed(),
		Handle:     handle,
		Report:     observed.report(),
		Now:        newDispatchClock().now,
	}, testPolicy())

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	if !published {
		t.Fatal("the message was supposed to be published before the recording failure")
	}
	if len(observed.failed) != 0 {
		t.Error("a PUBLISHED message must not be rescheduled as a publication failure")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventResolveFailed {
		t.Errorf("event = %q, want %q", got, domain.EventResolveFailed)
	}
}
