package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewDeactivate compose la fermeture d'un compte.
//
// # Ce que la fermeture atteint, et pourquoi c'est suffisant
//
// Elle ne supprime aucune session. Elle n'a pas à le faire : `Verify` relit
// l'identité à chaque appel et refuse une identité inactive, et `Grants` refuse
// de même. Le compte cesse donc de valoir au prochain appel, y compris pour les
// jetons déjà en circulation — c'est la décision 1 de l'ADR 017, et elle rend
// gratuit ce qui coûterait autrement un balayage du magasin.
//
// Supprimer les sessions en plus serait du nettoyage, pas de la sécurité. Le
// confondre ferait croire que le nettoyage est ce qui protège, et la protection
// disparaîtrait le jour où le nettoyage échouerait à moitié.
func NewDeactivate(deps Deps) ports.Deactivate {
	return deps.setActive(domain.Identity.Deactivated)
}

// NewReactivate compose la réouverture d'un compte.
func NewReactivate(deps Deps) ports.Reactivate {
	return deps.setActive(domain.Identity.Reactivated)
}

// setActive factorise les deux transitions.
//
// La transformation est passée en paramètre plutôt qu'un booléen : l'appelant
// nomme ce qu'il fait, et il n'existe aucun endroit où l'on puisse écrire
// « active = true » par inadvertance.
//
// Les deux sont IDEMPOTENTES : appliquer la même transition deux fois donne le
// même état. Deux administrateurs qui réagissent au même incident ne doivent pas
// s'annuler.
func (d Deps) setActive(transition func(domain.Identity) domain.Identity) func(
	context.Context, domain.IdentityID,
) error {
	return func(ctx context.Context, id domain.IdentityID) error {
		if id == "" {
			return fmt.Errorf("%w: l'identité est obligatoire", domain.ErrIncomplete)
		}

		identity, err := d.FindIdentity(ctx, id)
		if err != nil {
			return fmt.Errorf("identité: %w", err)
		}
		if err := d.UpdateIdentity(ctx, transition(identity)); err != nil {
			return fmt.Errorf("mise à jour de l'identité: %w", err)
		}
		return nil
	}
}
