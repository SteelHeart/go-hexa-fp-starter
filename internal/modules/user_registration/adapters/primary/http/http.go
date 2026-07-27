// Package http expose le module d'inscription sur la surface HTTP.
//
// # Une surface est un TRADUCTEUR, jamais une place métier
//
// Ce paquet fait exactement trois choses : traduire une requête HTTP en commande
// de domaine, appeler le cas d'usage, traduire le Result en réponse HTTP. Il ne
// valide rien lui-même — le domaine le fait déjà, et dupliquer la validation
// garantit qu'un jour les deux divergeront.
//
// Ajouter une surface (CLI, gRPC, consommateur d'événements) consiste à écrire
// un frère de ce fichier. Aucune ligne de `domain/` ni de `application/` ne
// bouge : c'est la propriété que l'ADR 005 achète.
package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// registerInput est la requête d'inscription.
//
// # Aucune contrainte de validation dans le schéma, et c'est délibéré
//
// La première version portait `format:"email"`. Le résultat, constaté en
// exécutant le serveur : huma refusait l'adresse AVANT le domaine, et rendait
// son propre message — « expected string to be RFC 5322 email » — en anglais,
// au milieu d'une API dont tous les autres messages viennent du domaine et sont
// en français.
//
// Deux règles de validation pour un même champ, c'est une de trop : elles
// divergeront, et c'est la mauvaise qui parlera à l'utilisateur. Le schéma
// DOCUMENTE donc (doc, example), il ne valide pas. `domain.NewEmail` reste seul
// juge, et c'est son message qui remonte.
type registerInput struct {
	Body struct {
		Email    string `json:"email"    doc:"Adresse de courriel" example:"alice@example.com"`
		Password string `json:"password" doc:"Mot de passe en clair. Jamais journalisé, jamais renvoyé."`
	}
}

// registerOutput porte la réponse d'inscription.
//
// Le corps est le type du LANGAGE PUBLIÉ, pas le type du domaine. Sérialiser
// `domain.User` exposerait le condensé du mot de passe le jour où quelqu'un
// ajoute un tag JSON — ici, c'est structurellement impossible.
type registerOutput struct {
	Status int
	Body   contract.RegisterResponse
}

// Mount enregistre les opérations du module sur l'API.
//
// Reçoit le Module, jamais un pilote ni un pool : une surface ne peut pas
// contourner le cas d'usage, même par accident.
func Mount(api huma.API, mod userregistration.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "register-user",
		Method:      contract.RegisterRoute.Method,
		Path:        contract.RegisterRoute.Path,
		Summary:     "Inscrire un utilisateur",
		Description: "Crée un compte en attente de confirmation. " +
			"Le compte naît `pending` : l'adresse n'est pas encore prouvée.",
		Tags:          []string{"users"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *registerInput) (*registerOutput, error) {
		command := domain.RegistrationCommand{
			Email:    in.Body.Email,
			Password: in.Body.Password,
		}

		value, failure, ok := mod.Register(ctx, command).Get()
		if !ok {
			return nil, statusFor(failure)
		}

		return &registerOutput{
			Status: http.StatusCreated,
			Body: contract.RegisterResponse{
				UserID:    value.ID.String(),
				Email:     value.Email.String(),
				Status:    string(value.Status),
				CreatedAt: value.CreatedAt,
			},
		}, nil
	})
}

// statusFor traduit une erreur de domaine en réponse HTTP.
//
// Le `switch` est EXHAUSTIF, et le linter `exhaustive` échoue la CI s'il ne
// l'est plus. C'est l'effet recherché : ajouter un code d'erreur au domaine
// force à décider, ici, ce que l'extérieur en voit. Sans cette contrainte, un
// nouveau code se traduirait silencieusement en 500 — et une erreur de saisie
// utilisateur serait comptée comme une panne de service.
//
// Le `default` reste, malgré l'exhaustivité : il couvre la valeur zéro et toute
// chaîne fabriquée par une conversion. Deny par défaut — l'inconnu est traité
// comme une panne, jamais comme une requête acceptable.
func statusFor(err domain.Error) error {
	switch err.Code {
	case domain.CodeInvalidEmail, domain.CodeWeakPassword:
		return huma.Error422UnprocessableEntity(err.Message, fieldDetail(err))

	case domain.CodeEmailAlreadyExists:
		// 409 et non 422 : la requête est bien formée, c'est l'ÉTAT du serveur
		// qui s'y oppose. La distinction compte pour un client qui décide de
		// réessayer ou non.
		return huma.Error409Conflict(err.Message, fieldDetail(err))

	case domain.CodeUnavailable:
		// 503 et non 500 : transitoire. C'est ce qui autorise un client à
		// réessayer, et ce qui évite de réveiller quelqu'un pour une base qui
		// redémarre.
		return huma.Error503ServiceUnavailable(err.Message)

	case domain.CodeInternal:
		// La cause technique n'est JAMAIS rendue à l'appelant : une erreur SQL
		// renvoyée au client est une fuite de structure. Elle est journalisée
		// par le middleware, qui la relie à l'identifiant de corrélation.
		return huma.Error500InternalServerError("une erreur interne est survenue")

	default:
		return huma.Error500InternalServerError("une erreur interne est survenue")
	}
}

// fieldDetail nomme le champ fautif quand le domaine l'a précisé.
//
// Sans lui, un formulaire ne saurait pas quel champ surligner et afficherait
// l'erreur en haut de page — ce qui, sur un formulaire à deux champs, oblige
// l'utilisateur à deviner.
func fieldDetail(err domain.Error) error {
	if err.Field == "" {
		return nil
	}
	return &huma.ErrorDetail{
		Location: "body." + err.Field,
		Message:  err.Message,
	}
}

// availabilityOutput porte la réponse de disponibilité.
type availabilityOutput struct {
	Body struct {
		Available bool `json:"available" doc:"Vrai si l'adresse est encore libre"`
	}
}

type availabilityInput struct {
	Email string `query:"email" required:"true" doc:"Adresse à vérifier"`
}

// MountAvailability expose la vérification de disponibilité d'une adresse.
//
// Séparée de Mount : c'est un port de LECTURE, et il n'a ni les mêmes besoins de
// cache, ni les mêmes besoins de limitation de débit qu'une écriture. Les monter
// ensemble ferait croire qu'ils se gouvernent pareil.
//
// ⚠️ NON-GARANTIE assumée : ce point d'entrée permet d'ÉNUMÉRER les adresses
// enregistrées. C'est acceptable tant qu'il est limité en débit ; ça ne l'est
// plus sans. Le module `ratelimit` n'existe pas encore — voir SECURITY.md.
func MountAvailability(api huma.API, mod userregistration.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "check-email-availability",
		Method:      http.MethodGet,
		Path:        "/v1/users/availability",
		Summary:     "Vérifier la disponibilité d'une adresse",
		Tags:        []string{"users"},
	}, func(ctx context.Context, in *availabilityInput) (*availabilityOutput, error) {
		available, failure, ok := mod.CheckEmail(ctx, in.Email).Get()
		if !ok {
			return nil, statusFor(failure)
		}

		out := &availabilityOutput{}
		out.Body.Available = available
		return out, nil
	})
}
