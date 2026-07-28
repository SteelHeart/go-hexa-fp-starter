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
//	http.go            Mount, et la carte des routes
//	open_session.go    POST   /v1/auth/sessions          — échanger un secret
//	close_session.go   DELETE /v1/auth/sessions/current  — révoquer
//	identity.go        GET    /v1/auth/identity          — résoudre le jeton
//	status.go          la traduction des refus en statuts, et le porteur
//
// # Ce que cette surface n'expose PAS, et pourquoi
//
// Ni inscription, ni définition de rôle, ni affectation, ni fermeture de compte.
// Ce sont des opérations d'ADMINISTRATION : les exposer sans les protéger
// ouvrirait à quiconque la création de comptes et l'attribution de droits, et les
// protéger exige un premier administrateur — donc une décision d'amorçage qui
// n'est pas prise (ADR 017 § ce qui n'est pas tranché).
//
// Elles restent joignables par le composition root, qui les tient déjà. Rien
// n'est perdu ; ce qui est refusé, c'est de les rendre publiques avant d'avoir
// tranché QUI a le droit de les appeler. Deny par défaut, y compris sur ce qu'on
// n'a pas encore décidé.
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
}
