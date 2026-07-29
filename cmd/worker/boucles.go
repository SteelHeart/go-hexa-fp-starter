package main

import (
	"context"
	"fmt"
)

// loop holds BOTH loops of the dispatcher until cancellation.
//
// # Why two and not one
//
// The dispatcher takes the messages out of the outbox and publishes them; the
// consumer receives the envelopes and triggers the effects. With the `inproc`
// relay both live in the same process, and one fills what the other empties.
//
// # The failure of one must stop the other
//
// Without that, a dead consumer would let the dispatcher publish to nobody —
// and a dispatcher publishing to nobody looks IN EVERY RESPECT like a
// dispatcher that works: the messages are marked published, the outbox drains,
// the metrics are green, and no effect takes place.
//
// The context is therefore cancelled at the first exit, whichever it is, and
// the error returned is the one that caused the shutdown.
func (w worker) loop(ctx context.Context) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	// A buffer of 2: both loops must be able to write even if nobody reads any
	// more. Without a buffer, the second would stay blocked forever on the
	// send, and the graceful shutdown would no longer be one.
	exits := make(chan error, 2)

	go func() {
		if err := w.dispatch.Run(ctx); err != nil {
			exits <- fmt.Errorf("dispatcher: %w", err)
			return
		}
		exits <- nil
	}()

	go func() {
		if err := w.consume.Run(ctx); err != nil {
			exits <- fmt.Errorf("event consumer: %w", err)
			return
		}
		exits <- nil
	}()

	// The FIRST exit decides: it cancels the context, which makes the other
	// loop exit. We still wait for both, so as not to leave while a
	// reservation is still open.
	first := <-exits
	stop()
	second := <-exits

	if first != nil {
		return first
	}
	return second
}
