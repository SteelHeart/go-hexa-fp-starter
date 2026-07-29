package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestPublishedEventIsAddressedAndVersioned: the event leaves with its versioned
// name and the identifier of its aggregate.
//
// The aggregate identifier is not decorative: it is what allows every event of a
// user to be found during an incident, and a sequence to be replayed in the
// right order. Without it, the outbox is nothing but a heap of messages.
//
// The content is checked too: the password digest has no business being in a
// message that will cross a broker.
func TestPublishedEventIsAddressedAndVersioned(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	_ = userOf(t, register(nominalDeps(observed)))

	if observed.eventType != domain.EventUserRegistered {
		t.Errorf("event type = %q, want %q", observed.eventType, domain.EventUserRegistered)
	}
	if observed.aggregate != identifier.String() {
		t.Errorf("aggregate identifier = %q, want %q", observed.aggregate, identifier)
	}

	encoded, err := json.Marshal(observed.event)
	if err != nil {
		t.Fatalf("the event must be serialisable: %v", err)
	}
	if strings.Contains(string(encoded), digest) {
		t.Errorf("the event carries the digest: %s", encoded)
	}
	if !strings.Contains(string(encoded), validAddress) {
		t.Error("the event must carry the address: the consumer needs it")
	}
}
