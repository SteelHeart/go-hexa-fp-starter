package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnEnvelopeSurvivesARoundTrip: the envelope crosses the transport intact.
//
// # Why this test exists
//
// The envelope is the PUBLISHED CONTRACT of the starter: it is what a consumer
// written in another language, or deployed separately, is going to read. Both
// network relays serialise it into JSON, and nothing else checks that passage.
//
// Three concrete pitfalls:
//
//   - A forgotten `json` tag renames the field to its Go name — `AggregateID`
//     instead of `aggregate_id`. The consumer in another language reads nil, and
//     the producer sees nothing.
//   - `Payload []byte` is encoded in base64 by `encoding/json`. That is correct,
//     but it MUST come back out identical: the payload is opaque JSON, we never
//     reinterpret it.
//   - `omitempty` on `traceparent`: an empty trace must not take the place of an
//     absent trace.
func TestAnEnvelopeSurvivesARoundTrip(t *testing.T) {
	t.Parallel()

	original := envelope("user.registered.v1")
	original.TraceParent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	original.Headers = map[string]string{"tenant": "acme"}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("serialisation of the envelope: %v", err)
	}

	// The field names ARE the contract: they are asserted literally, never by
	// reflection over the structure — otherwise the test would meekly follow a
	// rename that breaks every consumer.
	for _, field := range []string{
		`"id"`, `"type"`, `"aggregate_id"`, `"payload"`, `"traceparent"`,
		`"occurred_at"`, `"headers"`,
	} {
		if !strings.Contains(string(raw), field) {
			t.Errorf("field %s absent from the produced JSON: %s", field, raw)
		}
	}

	var back messaging.Envelope
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("deserialisation of the envelope: %v", err)
	}

	if back.ID != original.ID || back.Type != original.Type ||
		back.AggregateID != original.AggregateID {
		t.Errorf("identity altered by the round trip: %+v", back)
	}
	if !bytes.Equal(back.Payload, original.Payload) {
		t.Errorf("payload altered: %q, want %q", back.Payload, original.Payload)
	}
	if !back.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("timestamp altered: %s, want %s", back.OccurredAt, original.OccurredAt)
	}
	if back.TraceParent != original.TraceParent {
		t.Errorf("trace lost: %q — the correlation would stop at the broker", back.TraceParent)
	}
	if back.Headers["tenant"] != "acme" {
		t.Errorf("headers altered: %v", back.Headers)
	}
}

// TestAnAbsentTraceIsOmittedNotEmptied: no empty field in the contract.
//
// A transmitted `"traceparent":""` forces every consumer to tell the empty
// string apart from absence. That is exactly the kind of detail one can no
// longer take away once an external consumer has accommodated it.
func TestAnAbsentTraceIsOmittedNotEmptied(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(envelope("user.registered.v1"))
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}
	if strings.Contains(string(raw), `"traceparent"`) {
		t.Errorf("empty traceparent present in the JSON: %s", raw)
	}
	if strings.Contains(string(raw), `"headers"`) {
		t.Errorf("empty headers present in the JSON: %s", raw)
	}
}
