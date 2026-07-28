package tests

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// TestTheTraceIsRejoinedNotRestarted est le témoin du troisième critère de #9.
//
// # Ce que l'oubli produit, et pourquoi personne ne le remarque
//
// Tout continue de fonctionner. Les événements partent, les effets ont lieu, les
// tests passent. Seulement la moitié asynchrone d'une requête devient une trace
// ORPHELINE : l'inscription a la sienne, l'envoi du courriel a la sienne, et
// rien ne les relie.
//
// Le jour où un envoi échoue, on ne peut pas remonter à l'inscription qui l'a
// causé — et c'est précisément le jour où l'on en a besoin. C'est la même faute
// que `relay` documente en sens inverse : oublier `TraceParent` au dépilage
// coupe la chaîne côté producteur. Les deux moitiés doivent être branchées pour
// qu'une seule serve à quelque chose.
//
// Le test compare l'identifiant de trace REÇU par le gestionnaire à celui écrit
// dans l'enveloppe. Vérifier seulement qu'un contexte « a une trace » passerait
// avec une trace toute neuve — c'est-à-dire avec le défaut.
func TestTheTraceIsRejoinedNotRestarted(t *testing.T) {
	// Pas de `t.Parallel()` : ce test installe le propagateur GLOBAL, celui que
	// `telemetry.Setup` configure en production. En construire un local
	// éprouverait un propagateur que le producteur n'utilise pas.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	const (
		traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		spanID  = "00f067aa0ba902b7"
	)
	parent := "00-" + traceID + "-" + spanID + "-01"

	effet := &compteur{}
	env := envelope("evt-trace")
	env.TraceParent = parent

	if err := subscriber.WithTrace(effet.handler())(context.Background(), env); err != nil {
		t.Fatalf("le gestionnaire ne doit pas échouer : %v", err)
	}

	span := trace.SpanContextFromContext(effet.dernierContexte(t))
	if !span.IsValid() {
		t.Fatal("aucun contexte de trace restauré : la moitié asynchrone serait orpheline")
	}
	if span.TraceID().String() != traceID {
		t.Fatalf("trace RECOMMENCÉE au lieu d'être rejointe : %s au lieu de %s",
			span.TraceID(), traceID)
	}
	if span.SpanID().String() != spanID {
		t.Fatalf("span parent perdu : %s au lieu de %s", span.SpanID(), spanID)
	}
}

// TestAnEnvelopeWithoutTraceIsStillDelivered garde la remise avant la trace.
//
// Une enveloppe sans `traceparent` n'est PAS une erreur : le contexte d'origine
// repart intact et un nouveau tronçon commence. Refuser ferait perdre un
// événement pour un défaut d'observabilité — sacrifier ce qu'on mesure à la
// façon dont on le mesure.
//
// Le cas n'est pas théorique : tout producteur écrit avant le branchement de la
// télémétrie (#13) a laissé le champ vide.
func TestAnEnvelopeWithoutTraceIsStillDelivered(t *testing.T) {
	t.Parallel()

	effet := &compteur{}
	if err := subscriber.WithTrace(effet.handler())(context.Background(), envelope("evt-sans-trace")); err != nil {
		t.Fatalf("une enveloppe sans trace doit être remise : %v", err)
	}
	if effet.total() != 1 {
		t.Fatalf("l'effet devait avoir lieu, obtenu %d appels", effet.total())
	}
}

// TestAMalformedTraceParentDoesNotDropTheEnvelope garde la remise aussi.
//
// Un en-tête illisible ne doit pas coûter un événement. Le propagateur ignore ce
// qu'il ne comprend pas, et le gestionnaire est appelé quand même — la remise du
// message compte plus que la continuité de sa trace.
func TestAMalformedTraceParentDoesNotDropTheEnvelope(t *testing.T) {
	t.Parallel()

	for _, parent := range []string{"pas-un-traceparent", "00-", "99-xxx-yyy-zz"} {
		effet := &compteur{}
		env := envelope("evt-trace-cassee")
		env.TraceParent = parent

		if err := subscriber.WithTrace(effet.handler())(context.Background(), env); err != nil {
			t.Errorf("traceparent %q : l'enveloppe doit être remise, obtenu %v", parent, err)
		}
		if effet.total() != 1 {
			t.Errorf("traceparent %q : l'effet devait avoir lieu", parent)
		}
	}
}
