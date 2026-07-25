package tests

import (
	"context"
	"errors"
	"testing"

	outboxdomain "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
)

// TestPublicationFailureReachesTheDispatcher : un échec de publication remonte
// intact.
//
// Le recul exponentiel et l'abandon après N essais sont décidés par une
// politique PURE et testée, dans outbox/application. Ce pont n'a donc rien à
// décider.
//
// Les deux façons de se tromper ici sont graves et opposées :
//
//   - Avaler l'erreur ferait marquer le message comme TRAITÉ alors que rien
//     n'est parti. L'événement serait perdu définitivement, sans aucune trace —
//     le seul défaut que l'outbox existe précisément pour rendre impossible.
//   - La rattraper pour décider soi-même dupliquerait la politique de recul, et
//     les deux divergeraient au premier ajustement.
func TestPublicationFailureReachesTheDispatcher(t *testing.T) {
	t.Parallel()

	unreachable := errors.New("courtier injoignable")
	handle := relay.FromOutbox(func(context.Context, messaging.Envelope) error {
		return unreachable
	})

	err := handle(context.Background(), outboxdomain.Message{
		ID:   "019f9b46-3aec-735a-977d-129192ef130f",
		Type: "user.registered.v1",
	})

	if err == nil {
		t.Fatal("un échec de publication ne doit JAMAIS être avalé : le message serait marqué traité")
	}
	if !errors.Is(err, unreachable) {
		t.Errorf("erreur = %v, la cause doit remonter intacte au dépileur", err)
	}
}
