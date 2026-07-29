package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestAnUnknownTransportRefusesToResolve: deny by default on the transport.
//
// # The defect this test catches
//
// An unknown mode falling back on `inproc` — the cheapest mode, hence the
// "reasonable" fallback — would produce a LOCAL call where the operator asked
// for a remote one. The two modules are deployed separately, the calling module
// silently runs its own copy of the capability, and everything works: the data
// simply goes into the wrong database.
//
// This is the worst case of a silent fallback — it does not fail, it gets it
// wrong.
func TestAnUnknownTransportRefusesToResolve(t *testing.T) {
	t.Parallel()

	// Including the values that look like a valid mode: an intention is not
	// guessed from a typing mistake.
	for _, mode := range []string{"grpc", "HTTP", "in-proc", "events", "off"} {
		var localCalls int

		_, err := modulebus.Resolve(
			modulebus.New(interop(mode, nil), noPublisher(t)),
			someModule, route(), someEvent, localCaller(&localCalls))

		if err == nil {
			t.Errorf("mode=%q accepted — an unplanned transport must refuse resolution", mode)
			continue
		}
		if !errors.Is(err, modulebus.ErrUnknownMode) {
			t.Errorf("mode=%q returned with %v, want ErrUnknownMode", mode, err)
		}
	}
}
