// Package tests exercises the event relay through its PUBLIC API.
//
// What these tests do NOT cover, and it must be said: the `kafka` and
// `rabbitmq` relays open a network connection as soon as they are built. No
// broker exists here, so they stay WRITTEN, UNPROVEN — it is the
// `-tags=integration` level that will exercise them, in CI.
//
// Everything else — the choice of the relay, the in-memory bus, the silent
// relay, the retry decorator, the envelope — is tested without any
// infrastructure, and that is exactly what the starter promises.
package tests

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// quietLogger returns a silent logger, for the tests that do not observe it.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// recordingLogger returns a logger AND its buffer: some tests check that an
// abnormal situation was indeed SIGNALLED, not merely tolerated.
func recordingLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(handler), buf
}

// relayConfig builds a minimal configuration for a given relay.
func relayConfig(driver string) config.Messaging {
	return config.Messaging{
		Driver:         driver,
		TopicPrefix:    "hexa",
		ConsumerGroup:  "tests",
		PublishTimeout: config.Duration(time.Second),
	}
}

// envelope builds a test envelope of the given type.
func envelope(eventType string) messaging.Envelope {
	return messaging.Envelope{
		ID:          "01890000-0000-7000-8000-000000000000",
		Type:        eventType,
		AggregateID: "agg-1",
		Payload:     []byte(`{"k":"v"}`),
		OccurredAt:  time.Date(2026, time.July, 26, 10, 0, 0, 0, time.UTC),
	}
}

// mustBroker mounts a relay, or fails the test.
func mustBroker(t *testing.T, driver string) messaging.Broker {
	t.Helper()

	broker, err := messaging.New(relayConfig(driver), quietLogger())
	if err != nil {
		t.Fatalf("mounting of the %q relay failed: %v", driver, err)
	}
	return broker
}

// cancelled returns an already cancelled context, to wake up a blocking Run
// without waiting.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
