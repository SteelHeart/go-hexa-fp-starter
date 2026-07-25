package domain

import "time"

// UserID identifie un utilisateur. Type dédié, jamais une string nue dans une
// signature (rules/ports-et-contrats.md §3).
type UserID string

// String retourne l'identifiant.
func (id UserID) String() string { return string(id) }

// IsZero indique un identifiant non renseigné.
func (id UserID) IsZero() bool { return id == "" }

// Status énumère les états d'un compte. Ensemble fermé, switch exhaustif.
type Status string

// Les états d'un compte.
const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
)

// User est un utilisateur enregistré. Tous ses champs sont des types du domaine :
// il est structurellement impossible d'y placer une valeur invalide.
type User struct {
	ID           UserID
	Email        Email
	PasswordHash PasswordHash
	Status       Status
	CreatedAt    time.Time
}

// NewUser construit un utilisateur nouvellement inscrit.
//
// Le compte naît PENDING : un courriel de confirmation devra l'activer. Naître
// actif serait un « fail-open » — l'adresse n'est pas encore prouvée.
func NewUser(id UserID, email Email, hash PasswordHash, at time.Time) User {
	return User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		Status:       StatusPending,
		CreatedAt:    at.UTC(),
	}
}

// WithStatus retourne une copie portant le nouvel état. L'original n'est jamais
// modifié : le récepteur est une valeur, et `revive` refuserait un pointeur.
func (u User) WithStatus(status Status) User {
	u.Status = status
	return u
}

// CanAuthenticate indique si le compte peut ouvrir une session.
//
// Deny par défaut : tout état inconnu refuse. Ajouter un Status sans compléter
// ce switch fait échouer la CI (`exhaustive`).
func (u User) CanAuthenticate() bool {
	switch u.Status {
	case StatusActive:
		return true
	case StatusPending, StatusBlocked:
		return false
	default:
		return false
	}
}
