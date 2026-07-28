package http

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// Guard protège une opération par une permission.
//
// # Ce que ce type rend enfin joignable
//
// `Authorize` était prouvé par 39 tests et exposé par aucune surface : personne
// ne pouvait s'en servir depuis l'extérieur. Il n'a pas été écrit plus tôt
// délibérément — un garde sans route à protéger est du code mort, et ce dépôt
// vient d'en fermer trois. Il arrive avec ses premières routes.
//
// # Pourquoi un appel EXPLICITE et non un intergiciel
//
// Un intergiciel qui déposerait l'identité dans le contexte violerait la
// décision 5 de l'ADR 017 — *l'identité entre par la commande, jamais par le
// contexte*. Elle rendrait aussi INVISIBLE le fait qu'une opération dépend de
// qui appelle : on lit la signature d'un gestionnaire sans savoir s'il est
// protégé, et la protection disparaît le jour où quelqu'un déplace la route hors
// du groupe intergicié.
//
// Ici, chaque opération protégée commence par une ligne qui NOMME sa permission.
// Retirer la protection demande de supprimer cette ligne — visible en revue.
type Guard struct {
	Module auth.Module
}

// Require exige une permission et rend l'identité de l'appelant.
//
// # Les deux appels ne sont PAS fusionnés
//
// `Verify` puis `Authorize` : le jeton authentifie, il n'autorise pas. Un seul
// appel qui ferait les deux ramènerait les permissions dans le périmètre du
// jeton par la porte de derrière, et la prochaine optimisation naturelle serait
// de les y mettre pour de bon.
//
// # L'ordre des refus compte
//
// Un jeton invalide rend **401** avant toute question de permission : dire
// « permission refusée » à quelqu'un qui n'est pas authentifié lui apprendrait
// que la route existe et qu'un droit la garde. Un porteur authentifié mais sans
// le droit rend **403** — et cette distinction-là est utile, elle lui évite de
// se reconnecter en boucle pour un droit qu'il n'aura pas davantage.
func (g Guard) Require(
	ctx context.Context, header string, permission domain.Permission,
) (domain.Identity, error) {
	token, err := bearer(header)
	if err != nil {
		return domain.Identity{}, err
	}

	identity, err := g.Module.Verify(ctx, token)
	if err != nil {
		return domain.Identity{}, statusFor(err)
	}

	if err := g.Module.Authorize(ctx, identity.ID, permission); err != nil {
		return domain.Identity{}, statusFor(err)
	}
	return identity, nil
}

// mustPermission construit une permission au MONTAGE, pas à l'appel.
//
// Une permission mal formée est une faute de programmation, pas une condition
// d'exécution : elle ne dépend d'aucune entrée. La découvrir au premier appel
// protégé rendrait un 500 en production pour une chaîne fautive écrite des mois
// plus tôt — et la route paraîtrait fonctionner jusque-là.
//
// La panique est donc VOLONTAIRE et vit dans un adaptateur, jamais dans le cœur
// (rules/README.md : aucun `panic` dans `domain/`, `ports/`, `application/`).
// Elle survient au montage des routes, c'est-à-dire au démarrage, c'est-à-dire
// avant qu'un seul appelant ne soit servi.
func mustPermission(raw string) domain.Permission {
	permission, err := domain.NewPermission(raw)
	if err != nil {
		panic("permission mal formée au montage: " + raw + ": " + err.Error())
	}
	return permission
}
