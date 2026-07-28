package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// bearerInput porte le jeton présenté.
//
// Le jeton passe par l'en-tête `Authorization` et non par le corps ou l'URL :
// une URL se retrouve dans les journaux d'accès, l'historique du navigateur et
// l'en-tête `Referer` envoyé au premier site tiers visité ensuite. Un jeton dans
// une URL est un jeton publié.
type bearerInput struct {
	Authorization string `header:"Authorization" doc:"Jeton porteur, sous la forme: Bearer <jeton>"`
}

// closeSessionOutput ne porte rien.
//
// 204 et non 200 avec un corps « déconnecté » : il n'y a rien à dire, et un corps
// inventé donnerait à un client l'idée de le lire.
type closeSessionOutput struct {
	Status int
}

// mountCloseSession expose la révocation du jeton présenté.
func mountCloseSession(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "close-session",
		Method:      contract.CloseSessionRoute.Method,
		Path:        contract.CloseSessionRoute.Path,
		Summary:     "Fermer la session courante",
		Description: "Révoque le jeton présenté, IMMÉDIATEMENT. " +
			"Idempotent : fermer une session déjà fermée rend 204.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *bearerInput) (*closeSessionOutput, error) {
		token, err := bearer(in.Authorization)
		if err != nil {
			return nil, err
		}
		if err := mod.Revoke(ctx, token); err != nil {
			return nil, statusFor(err)
		}
		return &closeSessionOutput{Status: http.StatusNoContent}, nil
	})
}
