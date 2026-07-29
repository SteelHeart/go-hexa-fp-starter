package tests

import (
	"context"
	"testing"
	"time"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
)

// TestRelayCarriesEveryField: the message → envelope translation loses nothing.
//
// # Why this test exists despite the triviality of the code
//
// It is a field-to-field mapping, hence "obviously correct" — and that is
// exactly the kind of code where an omission does not show. Forgetting
// `Payload` would publish empty envelopes; forgetting `TraceParent` would cut
// the trace between producer and consumer.
//
// In both cases, the dispatcher would report `published`, the message would be
// marked as handled, and nothing would signal the loss. The defect would only
// be discovered at the end of the chain, at a consumer receiving nothing
// usable — and the whole journey would then have to be walked back to find it.
func TestRelayCarriesEveryField(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.July, 26, 8, 0, 0, 0, time.UTC)
	msg := outboxdomain.Message{
		ID:          "019f9b46-3aec-735a-977d-129192ef130f",
		Type:        "user.registered.v1",
		AggregateID: "user-42",
		Payload:     []byte(`{"user_id":"user-42"}`),
		TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Headers:     map[string]string{"tenant": "acme"},
		CreatedAt:   createdAt,
	}

	var captured messaging.Envelope
	handle := relay.FromOutbox(func(_ context.Context, env messaging.Envelope) error {
		captured = env
		return nil
	})

	if err := handle(context.Background(), msg); err != nil {
		t.Fatalf("relay: %v", err)
	}

	fields := map[string]struct{ got, want string }{
		"ID":          {captured.ID, msg.ID.String()},
		"Type":        {captured.Type, msg.Type},
		"AggregateID": {captured.AggregateID, msg.AggregateID},
		"Payload":     {string(captured.Payload), string(msg.Payload)},
		"TraceParent": {captured.TraceParent, msg.TraceParent},
	}
	for name, field := range fields {
		if field.got != field.want {
			t.Errorf("%s = %q, want %q", name, field.got, field.want)
		}
	}

	// OccurredAt carries the CREATION date, not that of the publication: a
	// consumer orders facts according to the moment they occurred. After a
	// dispatcher breakdown, the two differ by several hours.
	if !captured.OccurredAt.Equal(createdAt) {
		t.Errorf("OccurredAt = %v, want the creation date %v", captured.OccurredAt, createdAt)
	}
	if captured.Headers["tenant"] != "acme" {
		t.Errorf("Headers = %v, the tenant header is lost", captured.Headers)
	}
}
