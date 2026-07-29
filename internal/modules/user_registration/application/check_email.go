package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// NewCheckEmailAvailability builds the use case that checks whether an email
// address is available.
//
// This is a READ port: it is the one that receives the cache decorator. A cache
// on registration would make no sense at all — hence the existence of this
// second use case in the reference starter.
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
