package domain

import "time"

// EventUserRegistered nomme l'événement d'inscription.
//
// Le suffixe de version fait partie du nom : un consommateur déployé
// indépendamment continue de lire la v1 pendant que la v2 apparaît. Renommer un
// événement sans version est une rupture silencieuse.
const EventUserRegistered = "user.registered.v1"

// UserRegistered est l'événement publié après une inscription réussie.
//
// Il porte le strict nécessaire au consommateur. Y mettre l'utilisateur entier
// exposerait le condensé de mot de passe dans l'outbox — une table lue par des
// humains en incident.
type UserRegistered struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	RegisteredAt time.Time `json:"registered_at"`
}

// NewUserRegistered construit l'événement depuis l'utilisateur créé.
func NewUserRegistered(user User) UserRegistered {
	return UserRegistered{
		UserID:       user.ID.String(),
		Email:        user.Email.String(),
		RegisteredAt: user.CreatedAt,
	}
}
