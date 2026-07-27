package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// NewCheckEmailAvailability construit le cas d'usage de vérification de
// disponibilité d'une adresse.
//
// C'est un port de LECTURE : c'est lui qui reçoit le décorateur de cache. Un
// cache sur l'inscription n'aurait aucun sens — d'où l'existence de ce second
// cas d'usage dans le socle de référence.
func NewCheckEmailAvailability(emailIsTaken ports.EmailIsTaken) ports.CheckEmailAvailability {
	return func(ctx context.Context, rawEmail string) result.Result[bool, domain.Error] {
		return result.FlatMap(
			domain.NewEmail(rawEmail),
			func(email domain.Email) result.Result[bool, domain.Error] {
				return result.Map(
					emailIsTaken(ctx, email),
					func(taken bool) bool { return !taken },
				)
			},
		)
	}
}
