package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// openSessionInput est la demande de connexion.
//
// # Aucune contrainte de validation dans le schéma, et c'est délibéré
//
// Une borne de longueur ici ferait refuser la requête AVANT le domaine, avec le
// message de la bibliothèque plutôt que celui du module — et surtout, elle
// distinguerait « secret trop court » de « identifiants invalides ». Ce serait
// rouvrir l'oracle d'existence de comptes par la validation de schéma, après
// l'avoir fermé dans le cas d'usage.
//
// Le schéma DOCUMENTE, il ne valide pas.
type openSessionInput struct {
	Body struct {
		Subject string `json:"subject" doc:"Ce qui désigne le compte — adresse, identifiant" example:"alice@example.com"`
		Secret  string `json:"secret"  doc:"Secret en clair. Jamais journalisé, jamais renvoyé."`
	}
}

// openSessionOutput porte la session ouverte.
//
// Le corps est le type du LANGAGE PUBLIÉ, pas celui du domaine. Sérialiser une
// `domain.Session` exposerait ce que le domaine y ajouterait demain ; ici, ce qui
// sort est énuméré.
type openSessionOutput struct {
	Status int
	Body   contract.SessionResponse
}

// mountOpenSession expose l'échange d'un secret contre un jeton.
func mountOpenSession(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "open-session",
		Method:      contract.OpenSessionRoute.Method,
		Path:        contract.OpenSessionRoute.Path,
		Summary:     "Ouvrir une session",
		Description: "Échange un secret contre un jeton opaque. " +
			"Le jeton AUTHENTIFIE : il ne porte aucune permission, et les droits " +
			"sont relus à chaque décision (ADR 017).",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *openSessionInput) (*openSessionOutput, error) {
		session, err := mod.Authenticate(ctx, in.Body.Subject, in.Body.Secret)
		if err != nil {
			return nil, statusFor(err)
		}

		return &openSessionOutput{
			Status: http.StatusCreated,
			Body: contract.SessionResponse{
				Token:      session.Token.String(),
				IdentityID: string(session.Identity),
				ExpiresAt:  session.ExpiresAt,
			},
		}, nil
	})
}
