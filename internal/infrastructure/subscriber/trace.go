package subscriber

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// traceParentHeader est l'en-tête W3C que porte l'enveloppe.
const traceParentHeader = "traceparent"

// WithTrace reprend le contexte de trace déposé par le producteur.
//
// # Ce que l'oubli produit, et pourquoi personne ne le remarque
//
// Tout continue de fonctionner. Les événements partent, les effets ont lieu, les
// tests passent. Seulement, la moitié asynchrone d'une requête apparaît comme
// une trace ORPHELINE : l'inscription a sa trace, l'envoi du courriel a la
// sienne, et rien ne les relie. Le jour où un envoi échoue, on ne peut pas
// remonter à l'inscription qui l'a causé — et c'est précisément le jour où l'on
// en a besoin.
//
// C'est le même défaut que `relay` documente en sens inverse : oublier
// `TraceParent` au dépilage coupe la chaîne côté producteur. Les deux moitiés
// doivent être branchées pour qu'une seule des deux serve à quelque chose.
//
// # Une enveloppe SANS traceparent n'est pas une erreur
//
// Le contexte d'origine repart alors intact, et un nouveau tronçon commence.
// Refuser l'enveloppe ferait perdre un événement pour un défaut d'observabilité
// — sacrifier ce qu'on mesure à la façon dont on le mesure.
func WithTrace(handler messaging.Handler) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		return handler(restore(ctx, env), env)
	}
}

// restore extrait le contexte de trace de l'enveloppe.
//
// Passe par le propagateur GLOBAL plutôt que par une instance locale : c'est
// celui que `telemetry.Setup` configure, donc le même que le producteur a
// utilisé pour écrire. En construire un ici ferait diverger les deux moitiés le
// jour où la configuration changerait, sans qu'aucun test ne le voie.
func restore(ctx context.Context, env messaging.Envelope) context.Context {
	if env.TraceParent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{traceParentHeader: env.TraceParent}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
