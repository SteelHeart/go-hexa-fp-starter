package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewVerify compose la résolution d'un jeton en identité.
//
// # Trois causes, un seul refus
//
// Jeton inconnu, session expirée, identité désactivée : tout rend
// `domain.ErrTokenUnknown`. Répondre « expiré » plutôt qu'« inconnu » confirmerait
// qu'un tel jeton a existé, donc qu'un compte existe.
//
// # L'expiration est vérifiée ICI, pas seulement au magasin
//
// Un pilote peut purger paresseusement — celui en mémoire ne purge pas du tout.
// Compter sur lui pour ne rendre que des sessions valides ferait dépendre la
// sécurité d'un détail d'implémentation, et le premier pilote qui purgerait
// autrement rouvrirait la faille sans que rien ne le signale.
func NewVerify(deps Deps) ports.Verify {
	return func(ctx context.Context, token domain.Token) (domain.Identity, error) {
		if token.IsZero() {
			return domain.Identity{}, domain.ErrTokenUnknown
		}

		session, err := deps.FindSession(ctx, token)
		if err != nil {
			return domain.Identity{}, domain.ErrTokenUnknown
		}
		if session.Expired(deps.Now()) {
			return domain.Identity{}, domain.ErrTokenUnknown
		}

		identity, err := deps.FindIdentity(ctx, session.Identity)
		if err != nil || !identity.Active {
			return domain.Identity{}, domain.ErrTokenUnknown
		}
		return identity, nil
	}
}

// NewRevoke compose la révocation d'une session.
//
// Idempotente : révoquer un jeton déjà inconnu n'est pas une erreur. Un client
// qui se déconnecte deux fois n'a rien fait de mal, et faire échouer le second
// appel produirait une erreur que personne ne saurait traiter.
func NewRevoke(deps Deps) ports.Revoke {
	return func(ctx context.Context, token domain.Token) error {
		if token.IsZero() {
			return domain.ErrIncomplete
		}
		return deps.DeleteSession(ctx, token)
	}
}
