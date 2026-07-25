// Package relay branche le dépileur de l'outbox sur le transport d'événements.
//
// # Pourquoi ce paquet existe, et pourquoi ICI
//
// L'outbox est un module NOYAU : elle ne doit importer aucune infrastructure,
// sous peine de ne plus pouvoir être extraite en module Go indépendant
// (ADR 012). Le transport, lui, ignore tout de l'outbox — et c'est bien ainsi :
// il publie des enveloppes, sans savoir d'où elles viennent.
//
// Quelqu'un doit pourtant les relier. Ce paquet est ce quelqu'un : il consomme
// les deux et n'est consommé que par un composition root. La dépendance va donc
// de l'infrastructure vers le noyau, jamais l'inverse.
//
// Il vit hors de `cmd/` pour une raison précise : un mappage champ à champ
// paraît trivial et ne l'est pas. Oublier `Payload` publierait des enveloppes
// vides ; oublier `TraceParent` couperait la trace entre producteur et
// consommateur. Dans les deux cas le dépileur rapporterait `published`, le
// message serait marqué traité, et rien ne signalerait la perte. Ce code doit
// donc être testable — et du code dans `main` ne l'est qu'à moitié.
package relay

import (
	"context"
	"fmt"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	outboxports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// FromOutbox construit le gestionnaire que le dépileur appelle pour chaque
// message réservé.
//
// L'erreur de publication remonte TELLE QUELLE. C'est essentiel : le recul
// exponentiel et l'abandon après N essais sont décidés par une politique pure et
// testée, dans `outbox/application`. Avaler l'erreur ici ferait marquer le
// message comme traité alors que rien n'est parti — donc perdu définitivement,
// sans trace. Et la rattraper pour décider soi-même dupliquerait la politique,
// avec la certitude que les deux divergeront.
func FromOutbox(publish messaging.Publisher) outboxports.Handler {
	return func(ctx context.Context, msg outboxdomain.Message) error {
		if err := publish(ctx, envelopeOf(msg)); err != nil {
			return fmt.Errorf("publication de %s: %w", msg.Type, err)
		}
		return nil
	}
}

// envelopeOf traduit un message persisté en enveloppe de transport.
//
// `OccurredAt` porte la date de CRÉATION du message, pas celle de sa
// publication : un consommateur doit pouvoir ordonner les faits selon le moment
// où ils se sont produits, et non selon le moment où le dépileur les a sortis.
// Après une panne du dépileur, les deux diffèrent de plusieurs heures.
func envelopeOf(msg outboxdomain.Message) messaging.Envelope {
	return messaging.Envelope{
		ID:          msg.ID.String(),
		Type:        msg.Type,
		AggregateID: msg.AggregateID,
		Payload:     msg.Payload,
		TraceParent: msg.TraceParent,
		OccurredAt:  msg.CreatedAt,
		Headers:     msg.Headers,
	}
}
