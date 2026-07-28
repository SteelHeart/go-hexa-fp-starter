// Package auth est le LANGAGE PUBLIÉ du module d'authentification.
//
// # Ce que ce paquet contient, et ce qu'il ne contiendra jamais
//
// Des types primitifs sérialisables et des routes. Jamais un type du domaine,
// jamais une règle, jamais un accès aux données. `domain.Token`, `domain.Subject`
// et `domain.Permission` ont tous un champ PRIVÉ : les publier ici les rendrait
// fabricables de l'extérieur, et la normalisation qu'ils garantissent cesserait
// d'être une garantie.
//
// # Ce qui n'y figure pas, délibérément
//
// Aucune forme ne transporte de PERMISSION. C'est la décision 1 de l'ADR 017 vue
// depuis le contrat : le jeton authentifie, il n'autorise pas. Publier une
// réponse de connexion qui énumère les droits inviterait tout consommateur à les
// mettre en cache, et la révocation cesserait d'être immédiate sans qu'aucune
// ligne de ce dépôt n'ait changé.
//
// # Versionnement
//
// Un contrat est immuable. Une évolution cassante crée un `V2` à côté du `V1`.
package auth

import "time"

// ModuleName identifie le module propriétaire.
const ModuleName = "auth"

// SchemaName est le schéma Postgres du module.
//
// `platform` et non un schéma dédié : `auth` est un module NOYAU, et les modules
// noyau partagent le schéma de plateforme (ADR 011). Un schéma par module noyau
// multiplierait les rôles sans rien isoler de plus — ils appartiennent tous au
// socle.
const SchemaName = "platform"

// SessionRequest est la forme publiée d'une demande de connexion.
//
// `subject` et non `email` : le module ne présume pas de ce qui désigne un
// compte. Une adresse aujourd'hui, un identifiant externe demain, sans changer
// le contrat.
type SessionRequest struct {
	Subject string `json:"subject"`
	Secret  string `json:"secret"`
}

// SessionResponse est la forme publiée d'une session ouverte.
//
// # Trois champs, et pas un de plus
//
// Ni rôles, ni permissions, ni condensé. Un client reçoit de quoi s'authentifier
// et de quoi savoir quand recommencer — rien qui l'invite à décider lui-même de
// ce qu'il a le droit de faire.
//
// `expires_at` est une INFORMATION, pas une garantie : la session peut cesser de
// valoir avant, par révocation ou par fermeture du compte. Un client qui s'y
// fierait pour éviter de gérer un 401 se tromperait exactement le jour où ça
// compte.
type SessionResponse struct {
	Token      string    `json:"token"`
	IdentityID string    `json:"identity_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// IdentityResponse est la forme publiée d'une identité résolue.
//
// Les RÔLES y figurent, les permissions non. Un rôle est une étiquette
// d'administration — utile pour afficher « comptable » dans une interface — alors
// qu'une permission est une décision, et une décision se demande, elle ne se lit
// pas dans une réponse mise en cache.
type IdentityResponse struct {
	IdentityID string    `json:"identity_id"`
	Subject    string    `json:"subject"`
	Roles      []string  `json:"roles"`
	CreatedAt  time.Time `json:"created_at"`
}

// Routes exposées par la surface HTTP du module.
//
// Globales assumées : ce sont des constantes du langage publié, et Go n'a pas de
// constante structurée. Les rendre fonctions déguiserait une donnée en calcul
// sans rien protéger, puisque la valeur rendue serait de toute façon copiable.
//
//nolint:gochecknoglobals // constantes du langage publié : Go n'a pas de constante structurée
var (
	// OpenSessionRoute échange un secret contre un jeton.
	//
	// `POST /v1/auth/sessions` et non `/login` : la ressource est la SESSION, et
	// la créer est un POST. C'est ce qui rend la fermeture naturelle — un DELETE
	// sur la même ressource — au lieu d'un second verbe inventé.
	OpenSessionRoute = struct {
		Method string
		Path   string
	}{Method: "POST", Path: "/v1/auth/sessions"}

	// CloseSessionRoute révoque le jeton présenté.
	CloseSessionRoute = struct {
		Method string
		Path   string
	}{Method: "DELETE", Path: "/v1/auth/sessions/current"}

	// IdentityRoute résout le jeton présenté en identité.
	IdentityRoute = struct {
		Method string
		Path   string
	}{Method: "GET", Path: "/v1/auth/identity"}
)
