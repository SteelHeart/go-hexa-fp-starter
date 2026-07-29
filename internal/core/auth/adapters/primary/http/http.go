// Package http expose le module d'authentification sur la surface HTTP.
//
// # Une surface est un TRADUCTEUR, jamais une place métier
//
// Ce paquet traduit une requête en appel de cas d'usage, puis un retour en
// réponse. Il ne valide rien lui-même : le domaine le fait déjà, et dupliquer la
// validation garantit qu'un jour les deux divergeront — au détriment de celle qui
// parle à l'utilisateur.
//
// # Carte des fichiers
//
//	http.go             Mount, et la carte des routes
//	open_session.go     POST   /v1/auth/sessions          — échanger un secret
//	close_session.go    DELETE /v1/auth/sessions/current  — révoquer
//	identity.go         GET    /v1/auth/identity          — résoudre le jeton
//	guard.go            exiger une permission — le seul appelant d'Authorize
//	administration.go   les quatre routes PROTÉGÉES
//	status.go           la traduction des refus en statuts, et le porteur
//
// # Deux familles de routes, et la frontière entre elles est une permission
//
// Les trois routes de SESSION sont publiques par nécessité : on ne peut pas
// exiger un jeton de quelqu'un qui vient en demander un. Les quatre routes
// d'ADMINISTRATION exigent chacune une permission distincte.
//
// ⚠️ Ce paragraphe disait « cette surface n'expose PAS l'administration », faute
// d'un premier administrateur : les publier sans les protéger aurait ouvert la
// création de comptes à quiconque. L'amorçage tranché (ADR 017 §6), elles
// arrivent — avec le garde qui rend `Authorize` enfin joignable.
package http

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// apiTag regroupe les opérations du module dans le contrat servi.
//
// Une constante et non trois littéraux : un regroupement qui diverge d'une
// opération à l'autre éclate la section dans la documentation générée, et
// personne ne relit un contrat pour vérifier une étiquette.
const apiTag = "auth"

// Mount enregistre les opérations du module sur l'API.
//
// Reçoit le Module, jamais un pilote ni un magasin : une surface ne peut pas
// contourner un cas d'usage, même par accident.
//
// Les trois opérations sont montées ensemble, délibérément. Monter la connexion
// sans la révocation livrerait un service où l'on peut entrer sans pouvoir
// sortir — et la révocation est la propriété que l'ADR 017 achète.
func Mount(api huma.API, mod auth.Module) {
	mountOpenSession(api, mod)
	mountCloseSession(api, mod)
	mountIdentity(api, mod)
	mountAdministration(api, mod)
}
