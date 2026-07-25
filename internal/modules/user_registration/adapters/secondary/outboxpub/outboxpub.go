// Package outboxpub branche le port PublishEvent du module sur l'outbox du noyau.
//
// # Pourquoi ce paquet existe
//
// Le cas d'usage doit pouvoir publier un événement sans connaître l'outbox, et
// l'outbox doit rester un module noyau sans aucune notion de métier. Cet
// adaptateur est le seul point où les deux se rencontrent — c'est exactement le
// rôle d'un adaptateur secondaire.
//
// Il fait trois choses, et rien d'autre :
//
//  1. sérialise la charge utile en JSON (le noyau la traite comme opaque) ;
//  2. traduit l'erreur TECHNIQUE du noyau en erreur de DOMAINE ;
//  3. ne journalise rien — la cause est attachée à l'erreur, qui remonte.
//
// La traduction n'est pas une formalité : sans elle, une erreur de pilote
// remonterait jusqu'à la surface HTTP, qui la rendrait telle quelle à
// l'appelant. C'est une fuite de structure interne
// (rules/donnees-et-migrations.md §2).
package outboxpub

import (
	"context"
	"encoding/json"
	"errors"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	outboxports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// New construit le port de publication du module.
//
// `traceParent` est lu du contexte par l'appelant en amont ; il n'est pas
// reconstruit ici pour que cet adaptateur reste sans dépendance à la télémétrie.
func New(enqueue outboxports.Enqueue) ports.PublishEvent {
	return func(
		ctx context.Context,
		eventType string,
		aggregateID string,
		payload any,
	) result.Result[domain.Ack, domain.Error] {
		raw, err := json.Marshal(payload)
		if err != nil {
			// Un événement non sérialisable est un défaut de PROGRAMMATION, pas
			// une panne : il ne guérira pas au réessai. D'où CodeInternal, qui
			// produit un 500 — et non CodeUnavailable, qui inviterait le client
			// à réessayer indéfiniment une requête qui échouera toujours.
			return failure(domain.CodeInternal,
				"l'événement d'inscription n'a pas pu être préparé", err)
		}

		msg := outboxdomain.NewMessage{
			Type:        eventType,
			AggregateID: aggregateID,
			Payload:     raw,
		}

		if _, err := enqueue(ctx, msg); err != nil {
			return failure(translate(err),
				"l'inscription n'a pas pu être enregistrée", err)
		}
		return result.Ok[domain.Ack, domain.Error](domain.Ack{})
	}
}

// translate choisit le code de domaine correspondant à une panne technique.
//
// Un contexte annulé — client parti, délai dépassé — est TRANSITOIRE : il vaut
// CodeUnavailable, donc 503, donc réessayable. Le confondre avec CodeInternal
// ferait remonter une alerte de panne à chaque fois qu'un utilisateur ferme son
// onglet trop tôt, et cette alerte-là finit par être ignorée.
func translate(err error) domain.ErrorCode {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return domain.CodeUnavailable
	}
	return domain.CodeInternal
}

// failure construit l'erreur en attachant la cause.
//
// La cause est portée par l'erreur, jamais journalisée ici : un adaptateur qui
// journalise ET retourne produit deux traces pour un incident, et la seconde
// arrive sans identifiant de corrélation.
func failure(code domain.ErrorCode, message string, cause error) result.Result[domain.Ack, domain.Error] {
	return result.Err[domain.Ack, domain.Error](
		domain.NewError(code, message).WithCause(cause),
	)
}
