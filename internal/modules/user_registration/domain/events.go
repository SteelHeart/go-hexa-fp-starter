package domain

import "time"

// EventUserRegistered names the registration event.
//
// The version suffix is part of the name: a consumer deployed independently
// keeps reading v1 while v2 appears. Renaming an event without a version is a
// silent break.
const EventUserRegistered = "user.registered.v1"

// UserRegistered is the event published after a successful registration.
//
// It carries the strict minimum the consumer needs. Putting the whole user in it
// would expose the password digest in the outbox — a table read by humans during
// an incident.
type UserRegistered struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	RegisteredAt time.Time `json:"registered_at"`
}

// NewUserRegistered builds the event from the created user.
func NewUserRegistered(user User) UserRegistered {
	return UserRegistered{
		UserID:       user.ID.String(),
		Email:        user.Email.String(),
		RegisteredAt: user.CreatedAt,
	}
}
