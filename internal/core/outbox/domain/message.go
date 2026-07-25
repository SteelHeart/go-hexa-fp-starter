// Package domain porte le vocabulaire de l'outbox, sans aucune dépendance.
//
// Il n'importe ni pilote, ni driver de base, ni générateur d'identifiants :
// l'identifiant est fourni par le pilote, ce qui rend ce paquet testable sans
// rien et vérifiable par .arch-go.yml.
package domain

import "time"

// MessageID identifie un message de l'outbox.
type MessageID string

// String retourne l'identifiant.
func (id MessageID) String() string { return string(id) }

// Status énumère les états d'un message. Ensemble fermé : tout switch est
// vérifié exhaustif par le linter.
type Status string

// Les états d'un message.
const (
	// StatusPending attend sa publication, ou son prochain essai.
	StatusPending Status = "pending"
	// StatusDone a été publié avec succès.
	StatusDone Status = "done"
	// StatusFailed a épuisé ses tentatives. JAMAIS supprimé : c'est la seule
	// trace de ce qui n'a pas été publié.
	StatusFailed Status = "failed"
)

// NewMessage est une intention de publication, avant persistance.
//
// L'identifiant et l'horodatage n'y figurent pas : ce sont des effets, produits
// par le pilote. C'est ce qui rend un test déterministe.
type NewMessage struct {
	Type        string
	AggregateID string
	// Payload est du JSON opaque. Le domaine ne le sérialise pas lui-même : un
	// consommateur écrit dans un autre langage doit pouvoir le lire.
	Payload     []byte
	TraceParent string
	Headers     map[string]string
}

// Message est un message persisté.
type Message struct {
	ID          MessageID
	Type        string
	AggregateID string
	Payload     []byte
	TraceParent string
	Headers     map[string]string
	Status      Status
	Attempts    int
	CreatedAt   time.Time
	AvailableAt time.Time
}

// FailedAttempt décrit un échec de publication à enregistrer.
type FailedAttempt struct {
	ID          MessageID
	Attempts    int
	Status      Status
	AvailableAt time.Time
	Reason      string
}

// NextAttempt calcule l'état d'un message après un échec.
//
// Fonction PURE : ni horloge, ni aléa. L'instant courant est un paramètre, ce
// qui rend la politique de réessai testable sans attendre.
//
// Le recul est exponentiel et borné : sans borne, `1 << 40` produirait une durée
// négative après débordement, donc un message immédiatement rejoué en boucle.
func NextAttempt(msg Message, maxAttempts int, base time.Duration, now time.Time, reason string) FailedAttempt {
	attempts := msg.Attempts + 1

	status := StatusPending
	if attempts >= maxAttempts {
		status = StatusFailed
	}

	const maxShift = 10
	shift := attempts
	if shift > maxShift {
		shift = maxShift
	}

	return FailedAttempt{
		ID:          msg.ID,
		Attempts:    attempts,
		Status:      status,
		AvailableAt: now.Add(base * time.Duration(1<<shift)),
		Reason:      reason,
	}
}

// IsDue indique si un message doit être traité maintenant.
func (m Message) IsDue(now time.Time) bool {
	return m.Status == StatusPending && !m.AvailableAt.After(now)
}
