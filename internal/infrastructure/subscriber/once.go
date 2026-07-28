package subscriber

import (
	"context"
	"errors"
	"fmt"

	idempotencydomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	idempotencyports "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// Guard porte les trois ports de l'idempotence dont le décorateur a besoin.
//
// Une structure plutôt que trois paramètres : `Once(handler, reserve, complete,
// release)` fait quatre arguments, et surtout laisse l'appelant libre d'en
// inverser deux — `complete` et `release` ont la même signature, et l'inversion
// libérerait la clé de chaque succès tout en verrouillant chaque échec. Le
// compilateur ne verrait rien.
type Guard struct {
	Reserve  idempotencyports.Reserve
	Complete idempotencyports.Complete
	Release  idempotencyports.Release
}

// Once fait que l'effet n'a lieu qu'UNE fois, même si l'enveloppe revient.
//
// # Le protocole, et pourquoi chaque branche est là
//
//	réserver → déjà rejoué ? acquitter sans rien faire
//	         → en vol ?      RENDRE UNE ERREUR pour que le transport rejoue
//	         → sinon         exécuter, puis compléter ou libérer
//
// **Un rejeu acquitte sans exécuter.** C'est tout l'objet : les transports
// d'ici sont « au moins une fois », donc la même enveloppe arrive deux fois dès
// qu'un accusé de réception se perd. Sans ce garde, un courriel de bienvenue
// part deux fois — et le jour où le consommateur débitera une carte, ce sera le
// débit.
//
// **Un « en vol » rend une ERREUR, délibérément.** Une autre réplique traite
// déjà cette enveloppe et peut encore échouer. L'acquitter ici perdrait
// l'événement si l'autre abandonne. Rejouer plus tard ne coûte qu'une tentative ;
// acquitter à tort coûte l'effet.
//
// **`Release` est en `defer` sur le chemin d'échec.** L'oublier rendrait
// l'enveloppe intraitable jusqu'à l'expiration de la clé — soit vingt-quatre
// heures pendant lesquelles un rejeu est refusé sans que rien n'explique
// pourquoi.
func Once(guard Guard, handler messaging.Handler) messaging.Handler {
	return func(ctx context.Context, env messaging.Envelope) error {
		if env.ID == "" {
			return fmt.Errorf("%w: type=%q", ErrMissingID, env.Type)
		}

		key := idempotencydomain.Key(env.ID)
		reservation, err := guard.Reserve(ctx, idempotencydomain.Request{
			Key: key,
			// L'empreinte est le TYPE et non la charge utile : deux enveloppes
			// de même identifiant portent le même fait. Empreinter la charge
			// utile ferait échouer en conflit un rejeu que le producteur aurait
			// sérialisé différemment — un ordre de clés JSON suffit.
			Fingerprint: env.Type,
		})
		switch {
		case errors.Is(err, idempotencydomain.ErrInFlight):
			return fmt.Errorf("enveloppe %s déjà en cours de traitement: %w", env.ID, err)
		case err != nil:
			return fmt.Errorf("réservation de %s: %w", env.ID, err)
		case reservation.Replayed:
			// Déjà traité. On acquitte sans réexécuter : c'est la seule branche
			// qui rend `nil` sans avoir rien fait, et c'est la raison d'être du
			// paquet.
			return nil
		}

		if err := handler(ctx, env); err != nil {
			// Libérer AVANT de remonter, pour qu'un rejeu soit possible tout de
			// suite. L'erreur de libération est jointe plutôt qu'écrasée : une
			// clé restée verrouillée est un incident distinct, qui se
			// diagnostique mal quand son message a disparu.
			return errors.Join(
				fmt.Errorf("traitement de %s: %w", env.ID, err),
				release(ctx, guard, key),
			)
		}

		if err := guard.Complete(ctx, key, nil); err != nil {
			// L'effet A EU LIEU. Remonter l'erreur fera rejouer l'enveloppe, et
			// le rejeu ne sera PAS reconnu puisque la clé n'est pas complétée :
			// l'effet aura donc lieu deux fois. C'est le seul cas où ce paquet
			// ne tient pas sa promesse, et il est nommé plutôt que tu — même
			// forme que le « publié mais non marqué » du dépileur d'outbox.
			return fmt.Errorf("enveloppe %s traitée mais NON marquée: %w", env.ID, err)
		}
		return nil
	}
}

// release libère la clé et enveloppe son erreur.
func release(ctx context.Context, guard Guard, key idempotencydomain.Key) error {
	if err := guard.Release(ctx, key); err != nil {
		return fmt.Errorf("libération de la clé %s: %w", key, err)
	}
	return nil
}
