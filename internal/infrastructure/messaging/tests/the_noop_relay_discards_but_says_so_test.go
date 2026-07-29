package tests

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheNoopRelayDiscardsButSaysSo: the silent relay discards, and SAYS SO.
//
// # The defect this test catches
//
// A relay that discards the events WITHOUT SAYING ANYTHING is
// indistinguishable from a broken relay. It is the only driver of the starter
// whose nominal behaviour is loss: it must therefore be the most talkative, not
// the most silent.
//
// The trace is at Debug deliberately — in production the `noop` relay has no
// business being there, and requiring it at Info would drown the logs of a test
// environment. But it must NAME the event type: "publication discarded"
// without the name helps nobody understand why their email did not go out.
func TestTheNoopRelayDiscardsButSaysSo(t *testing.T) {
	t.Parallel()

	logger, logs := recordingLogger(slog.LevelDebug)

	broker, err := messaging.New(relayConfig(string(messaging.DriverNoop)), logger)
	if err != nil {
		t.Fatalf("mounting of the noop relay failed: %v", err)
	}

	// A consumer registered on the silent relay must receive NOTHING: that is
	// what makes it usable during a migration without a side effect.
	var received int
	broker.Consume.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		received++
		return nil
	})

	if err := broker.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("the silent relay returned an error: %v", err)
	}
	if received != 0 {
		t.Errorf("the silent relay delivered %d event(s) — it must deliver nothing", received)
	}

	trace := logs.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Errorf("the trace does not name the discarded type, trace=%q", trace)
	}
}

// TestTheNoopConsumerReturnsOnShutdown: Run hands back control on cancellation.
//
// A consumer that did not watch the context would block the shutdown of the
// worker until SIGKILL — including for the driver that, by definition, has
// nothing to finish.
func TestTheNoopConsumerReturnsOnShutdown(t *testing.T) {
	t.Parallel()

	broker := mustBroker(t, string(messaging.DriverNoop))

	done := make(chan error, 1)
	go func() { done <- broker.Consume.Run(cancelled()) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v on cancellation, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not hand back control on a cancelled context — the shutdown of the worker would block")
	}
}
