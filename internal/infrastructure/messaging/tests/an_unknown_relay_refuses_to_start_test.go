package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnUnknownRelayRefusesToStart: deny by default on the choice of transport.
//
// # The defect this test catches
//
// An unknown relay that fell back on "noop" would produce the worst possible
// defect of this starter: the service starts, answers, the outbox fills up, the
// dispatcher empties it, everything is marked as published — and NO event goes
// out. The welcome email never leaves, and nothing, nowhere, signals it.
//
// A typo in `messaging.driver` is enough. It must cost a noisy refusal to
// start, never a silent loss.
func TestAnUnknownRelayRefusesToStart(t *testing.T) {
	t.Parallel()

	// Including values that "look alike": a fallback on the closest one would
	// be an interpretation, and one does not interpret a configuration.
	for _, driver := range []string{"", "kafka2", "Inproc", "in-proc", "amqp", "none"} {
		if _, err := messaging.New(relayConfig(driver), quietLogger()); err == nil {
			t.Errorf("driver=%q accepted — an unplanned relay must refuse to start", driver)
		} else if !errors.Is(err, messaging.ErrUnknownDriver) {
			t.Errorf("driver=%q returned with %v, want ErrUnknownDriver", driver, err)
		}
	}
}
