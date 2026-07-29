package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
)

// TestAMalformedMessageNeverReachesTheProvider guards the UPSTREAM refusal.
//
// # Why the use case revalidates what the domain already knows
//
// `domain.Message` has EXPORTED fields: a caller can therefore build one without
// going through `NewMessage`, and nothing in the type prevents it. The use case
// is the last place where one can still refuse before an empty address reaches a
// provider — where it would become a billed rejection logged at a third party.
//
// The test deliberately builds messages BY HAND, exactly as a hurried caller
// would.
func TestAMalformedMessageNeverReachesTheProvider(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, logs := newModule(t, nil)

	valid := message(t)
	cases := map[string]domain.Message{
		"no recipient":       {Channel: domain.ChannelEmail, Subject: subject, Body: body},
		"no subject":         {Channel: domain.ChannelEmail, To: valid.To, Body: body},
		"whitespace subject": {Channel: domain.ChannelEmail, To: valid.To, Subject: "   ", Body: body},
		"no body":            {Channel: domain.ChannelEmail, To: valid.To, Subject: subject},
		"whitespace body":    {Channel: domain.ChannelEmail, To: valid.To, Subject: subject, Body: "  \n "},
		"bare message":       {},
	}

	for name, msg := range cases {
		err := mod.Send(ctx, msg)
		if !errors.Is(err, domain.ErrIncomplete) && !errors.Is(err, domain.ErrUnknownChannel) {
			t.Errorf("%s: want a domain refusal, got %v", name, err)
		}
	}

	if logs.text() != "" {
		t.Fatalf("no refused message must reach the driver: %s", logs.text())
	}
}

// TestAnUnservedChannelIsRefusedNeverSubstituted: deny by default on the
// channel.
//
// Falling back to email because SMS is not shipped would send a verification
// code to the wrong address — a channel chosen precisely because the other one
// did not suit. The catalogue says what EXISTS, and the domain refuses the rest.
func TestAnUnservedChannelIsRefusedNeverSubstituted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, logs := newModule(t, nil)
	valid := message(t)

	for _, channel := range []domain.Channel{"sms", "push", "webhook", "", "EMAIL"} {
		err := mod.Send(ctx, domain.Message{
			Channel: channel, To: valid.To, Subject: subject, Body: body,
		})
		if !errors.Is(err, domain.ErrUnknownChannel) {
			t.Errorf("channel %q: want ErrUnknownChannel, got %v", channel, err)
		}
	}

	if logs.text() != "" {
		t.Fatalf("no refused channel must reach the driver: %s", logs.text())
	}
}

// TestRecipientIsNormalisedSoDeduplicationHolds guards the canonical form.
//
// `Alice@Example.COM ` and `alice@example.com` are ONE address. Without
// normalisation, a send deduplication would let the duplicate through — and the
// recipient would receive the same welcome email twice, which is exactly the
// symptom idempotency is meant to remove.
func TestRecipientIsNormalisedSoDeduplicationHolds(t *testing.T) {
	t.Parallel()

	for _, variant := range []string{"  Alice@Example.COM ", "ALICE@EXAMPLE.COM", "alice@example.com"} {
		to, err := domain.NewRecipient(variant)
		if err != nil {
			t.Fatalf("address %q refused: %v", variant, err)
		}
		if to.String() != recipient {
			t.Errorf("address %q normalised to %q, want %q", variant, to.String(), recipient)
		}
	}
}

// TestAMalformedRecipientIsRefused guards the bounds of the address.
//
// The validation is deliberately MINIMAL — no "RFC 5322 compliant" regular
// expression. They reject valid addresses (apostrophes, accents, `+`
// sub-addresses), and a recipient wrongly refused never receives anything
// without anyone noticing. This test therefore guards the INDISPUTABLE refusals,
// and checks that exotic forms pass.
func TestAMalformedRecipientIsRefused(t *testing.T) {
	t.Parallel()

	refused := []string{"", "   ", "no-at-sign", "@example.com", "alice@", "a@b@c.com", "ali ce@x.com"}
	for _, raw := range refused {
		if _, err := domain.NewRecipient(raw); !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("address %q: want ErrIncomplete, got %v", raw, err)
		}
	}

	accepted := []string{"alice+facture@example.com", "l'éve@example.com", "a@b.co", "prénom.nom@sous.domaine.fr"}
	for _, raw := range accepted {
		if _, err := domain.NewRecipient(raw); err != nil {
			t.Errorf("valid address %q refused: %v", raw, err)
		}
	}
}
