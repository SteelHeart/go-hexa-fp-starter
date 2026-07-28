package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// secretMinLen borne la longueur d'un secret.
//
// Douze caractères, et aucune exigence de composition — ni majuscule, ni chiffre,
// ni caractère spécial. La longueur est la seule contrainte qui augmente
// réellement l'entropie ; les règles de composition poussent surtout à écrire
// `Password1!` et à le coller sous le clavier.
const secretMinLen = 12

// NewRegister compose la création d'une identité et de son secret.
//
// # L'ordre des étapes est la décision
//
// Valider la forme, hacher, écrire. Hacher AVANT d'écrire n'est pas un détail :
// c'est ce qui garantit que le secret en clair ne traverse jamais la frontière du
// magasin. Un pilote ne reçoit qu'un condensé, donc il ne peut pas le journaliser
// même par accident.
//
// L'unicité du sujet est tranchée par le MAGASIN, pas ici : entre une
// vérification et une écriture, deux demandes simultanées passent toutes les deux.
func NewRegister(deps Deps) ports.Register {
	return func(ctx context.Context, rawSubject, secret string) (domain.Identity, error) {
		subject, err := domain.NewSubject(rawSubject)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("sujet: %w", err)
		}
		if len([]rune(secret)) < secretMinLen {
			return domain.Identity{}, fmt.Errorf(
				"%w: le secret doit faire au moins %d caractères", domain.ErrIncomplete, secretMinLen)
		}

		hash, err := deps.HashSecret(secret)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("hachage du secret: %w", err)
		}

		identity, err := domain.NewIdentity(deps.NewIdentityID(), subject, nil, deps.Now())
		if err != nil {
			return domain.Identity{}, fmt.Errorf("création de l'identité: %w", err)
		}

		credential, err := domain.NewCredential(identity, hash)
		if err != nil {
			return domain.Identity{}, fmt.Errorf("assemblage de la créance: %w", err)
		}
		if err := deps.SaveIdentity(ctx, credential); err != nil {
			return domain.Identity{}, fmt.Errorf("enregistrement de l'identité: %w", err)
		}
		return identity, nil
	}
}
