package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAMountedRelayIsNeverHalfBuilt: the three faces of a Broker are always there.
//
// # Why this test exists
//
// `New` used to return four separate values. The caller could forget one —
// typically Close — and the omission only shows up the moment connections to
// the broker leak in production, weeks later.
//
// The `Broker` type removed that possibility, but only if EVERY relay fills the
// three fields. A relay without an external resource is precisely the one we
// are tempted to leave with a nil Close: this test forbids it, because the
// caller must never have to test before calling.
func TestAMountedRelayIsNeverHalfBuilt(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{
		string(messaging.DriverInproc),
		string(messaging.DriverNoop),
	} {
		broker := mustBroker(t, driver)

		if broker.Publish == nil {
			t.Errorf("%s: Publish is nil", driver)
		}
		if broker.Consume == nil {
			t.Errorf("%s: Consume is nil", driver)
		}
		if broker.Close == nil {
			t.Fatalf("%s: Close is nil — the caller must not have to test", driver)
		}
		if err := broker.Close(); err != nil {
			t.Errorf("%s: Close of a relay without an external resource returned %v", driver, err)
		}
	}
}
