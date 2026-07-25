// Package userregistration est le LANGAGE PUBLIÉ du module d'inscription.
//
// # Pourquoi ce paquet existe hors des features
//
// Une feature n'importe JAMAIS une autre feature (ADR 001) : cette règle est
// absolue et vérifiée par arch-go. Mais un module a parfois besoin d'une
// capacité d'un autre. Le contrat publié résout la contradiction : il est
// physiquement séparé des features, ne contient que des types primitifs, et
// n'expose RIEN du domaine interne.
//
// Ce que ce paquet peut contenir : des noms d'événements, des charges utiles
// sérialisables, des formes de commande et de réponse.
//
// Ce qu'il ne contiendra jamais : un type du domaine, une règle métier, un accès
// aux données. Le module producteur reste seul propriétaire de ses tables.
//
// # Versionnement
//
// Un contrat est immuable. Une évolution cassante crée un `V2` à côté du `V1` :
// les consommateurs sont déployés indépendamment et lisent encore le `V1`.
package userregistration

import "time"

// ModuleName identifie le module propriétaire. Sert de clé de configuration du
// transport et de nom de schéma Postgres.
const ModuleName = "user_registration"

// SchemaName est le schéma Postgres exclusif du module.
//
// Aucun autre module n'a le droit d'y accéder, ni en lecture. C'est vérifié par
// la garde CI `isolation` (ADR 011).
const SchemaName = "user_registration"

// EventUserRegisteredV1 nomme l'événement d'inscription publié.
const EventUserRegisteredV1 = "user.registered.v1"

// UserRegisteredV1 est la charge utile publiée à l'inscription d'un utilisateur.
//
// Types primitifs uniquement : un consommateur écrit dans un autre langage, ou
// déployé séparément, doit pouvoir la lire.
type UserRegisteredV1 struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	RegisteredAt time.Time `json:"registered_at"`
}

// RegisterRequest est la forme publiée de la commande d'inscription.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse est la forme publiée du résultat d'inscription.
//
// Ni condensé de mot de passe, ni champ interne : ce qui sort du module est ce
// qu'un autre module a le droit de connaître, rien de plus.
type RegisterResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRoute est la route HTTP qui expose la capacité, utilisée quand le
// module est appelé à distance.
var RegisterRoute = struct {
	Method string
	Path   string
}{Method: "POST", Path: "/v1/users"}
