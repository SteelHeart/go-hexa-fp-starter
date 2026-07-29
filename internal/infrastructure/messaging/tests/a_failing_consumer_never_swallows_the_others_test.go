package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAFailingConsumerNeverSwallowsTheOthers: a failure short-circuits nothing,
// and it GOES BACK UP.
//
// # The two defects this test catches
//
//  1. A `return err` inside the loop over the consumers. The first failure would
//     prevent the following ones from running — and since the iteration order is
//     that of the subscription, which module is deprived would depend on the
//     mounting order. A defect that changes victim at every restart.
//  2. A swallowed failure. The dispatcher would mark the message as published
//     although a consumer has not handled it: definitive loss, without a trace,
//     and the outbox would have done all its work for nothing.
//
// The error that goes back up must ALSO stay identifiable by `errors.Is`: the
// dispatcher tells failures apart to decide on the backoff, it does not read
// strings.
func TestAFailingConsumerNeverSwallowsTheOthers(t *testing.T) {
	t.Parallel()

	boom := errors.New("consumer unavailable")
	bus := messaging.NewInproc(quietLogger())

	var afterFailure int
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		return boom
	})
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		afterFailure++
		return nil
	})

	err := bus.Publish(context.Background(), envelope("user.registered.v1"))

	if err == nil {
		t.Fatal("a failing consumer did not fail the publication — " +
			"the dispatcher would mark the message as handled, and it would be lost")
	}
	if !errors.Is(err, boom) {
		t.Errorf("the error that went back up = %v, it must stay identifiable by errors.Is", err)
	}
	if afterFailure != 1 {
		t.Errorf("the following consumer was called %d times, want 1 — "+
			"a failure must not deprive the others of the event", afterFailure)
	}
}
