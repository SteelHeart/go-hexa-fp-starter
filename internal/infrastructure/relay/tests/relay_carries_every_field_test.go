package tests

import (
	"context"
	"testing"
	"time"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
)

// TestRelayCarriesEveryField : la traduction message → enveloppe ne perd rien.
//
// # Pourquoi ce test existe malgré la trivialité du code
//
// C'est un mappage champ à champ, donc « évidemment correct » — et c'est
// exactement le genre de code où une omission ne se voit pas. Oublier `Payload`
// publierait des enveloppes vides ; oublier `TraceParent` couperait la trace
// entre producteur et consommateur.
//
// Dans les deux cas, le dépileur rapporterait `published`, le message serait
// marqué traité, et rien ne signalerait la perte. Le défaut ne se découvrirait
// qu'au bout de la chaîne, chez un consommateur qui ne reçoit rien
// d'exploitable — et il faudrait alors remonter tout le trajet pour le trouver.
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
		t.Fatalf("relais: %v", err)
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
			t.Errorf("%s = %q, attendu %q", name, field.got, field.want)
		}
	}

	// OccurredAt porte la date de CRÉATION, pas celle de la publication : un
	// consommateur ordonne les faits selon le moment où ils se sont produits.
	// Après une panne du dépileur, les deux diffèrent de plusieurs heures.
	if !captured.OccurredAt.Equal(createdAt) {
		t.Errorf("OccurredAt = %v, attendu la date de création %v", captured.OccurredAt, createdAt)
	}
	if captured.Headers["tenant"] != "acme" {
		t.Errorf("Headers = %v, l'en-tête tenant est perdu", captured.Headers)
	}
}
