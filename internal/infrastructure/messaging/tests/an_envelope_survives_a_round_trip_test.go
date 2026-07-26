package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnEnvelopeSurvivesARoundTrip : l'enveloppe traverse le transport intacte.
//
// # Pourquoi ce test existe
//
// L'enveloppe est le CONTRAT PUBLIÉ du socle : c'est ce qu'un consommateur écrit
// dans un autre langage, ou déployé séparément, va lire. Les deux relais réseau
// la sérialisent en JSON, et rien d'autre ne vérifie ce passage.
//
// Trois pièges concrets :
//
//   - Une étiquette `json` oubliée renomme le champ en son nom Go — `AggregateID`
//     au lieu de `aggregate_id`. Le consommateur d'un autre langage lit nil, et
//     le producteur, lui, ne voit rien.
//   - `Payload []byte` est encodé en base64 par `encoding/json`. C'est correct,
//     mais il DOIT ressortir identique : la charge utile est du JSON opaque, on
//     ne la réinterprète jamais.
//   - `omitempty` sur `traceparent` : une trace vide ne doit pas occuper la place
//     d'une trace absente.
func TestAnEnvelopeSurvivesARoundTrip(t *testing.T) {
	t.Parallel()

	original := envelope("user.registered.v1")
	original.TraceParent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	original.Headers = map[string]string{"tenant": "acme"}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("sérialisation de l'enveloppe: %v", err)
	}

	// Les noms de champs SONT le contrat : ils s'affirment en dur, jamais par
	// réflexion sur la structure — sinon le test suivrait docilement un
	// renommage qui casse tous les consommateurs.
	for _, field := range []string{
		`"id"`, `"type"`, `"aggregate_id"`, `"payload"`, `"traceparent"`,
		`"occurred_at"`, `"headers"`,
	} {
		if !strings.Contains(string(raw), field) {
			t.Errorf("champ %s absent du JSON produit: %s", field, raw)
		}
	}

	var back messaging.Envelope
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("désérialisation de l'enveloppe: %v", err)
	}

	if back.ID != original.ID || back.Type != original.Type ||
		back.AggregateID != original.AggregateID {
		t.Errorf("identité altérée par l'aller-retour: %+v", back)
	}
	if !bytes.Equal(back.Payload, original.Payload) {
		t.Errorf("charge utile altérée: %q, attendu %q", back.Payload, original.Payload)
	}
	if !back.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("horodatage altéré: %s, attendu %s", back.OccurredAt, original.OccurredAt)
	}
	if back.TraceParent != original.TraceParent {
		t.Errorf("trace perdue: %q — la corrélation s'arrêterait au broker", back.TraceParent)
	}
	if back.Headers["tenant"] != "acme" {
		t.Errorf("en-têtes altérés: %v", back.Headers)
	}
}

// TestAnAbsentTraceIsOmittedNotEmptied : pas de champ vide dans le contrat.
//
// Un `"traceparent":""` transmis force chaque consommateur à distinguer la chaîne
// vide de l'absence. C'est exactement le genre de détail qu'on ne peut plus
// retirer une fois qu'un consommateur externe s'en est accommodé.
func TestAnAbsentTraceIsOmittedNotEmptied(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(envelope("user.registered.v1"))
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	if strings.Contains(string(raw), `"traceparent"`) {
		t.Errorf("traceparent vide présent dans le JSON: %s", raw)
	}
	if strings.Contains(string(raw), `"headers"`) {
		t.Errorf("headers vides présents dans le JSON: %s", raw)
	}
}
