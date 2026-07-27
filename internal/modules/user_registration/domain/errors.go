// Package domain porte les règles métier de l'inscription, sous forme de
// fonctions pures et de valeurs immuables.
//
// Ce paquet n'importe NI transport, NI persistance, NI logger. Il ne lit pas
// l'horloge et ne génère pas d'aléa : ces effets sont des ports, injectés.
// Vérifié par arch-go.yml et depguard.
package domain

// ErrorCode énumère les issues d'erreur possibles de la feature.
//
// L'ensemble est FERMÉ : tout switch sur ErrorCode est vérifié exhaustif par le
// linter `exhaustive`. Ajouter un code force donc à traiter sa traduction dans
// toutes les surfaces — c'est l'effet recherché.
type ErrorCode string

// Les codes d'erreur de la feature.
const (
	CodeInvalidEmail       ErrorCode = "invalid_email"
	CodeWeakPassword       ErrorCode = "weak_password"
	CodeEmailAlreadyExists ErrorCode = "email_already_exists"
	CodeUnavailable        ErrorCode = "unavailable"
	CodeInternal           ErrorCode = "internal"
)

// Error est une erreur métier. C'est une VALEUR, pas une interface : le cœur ne
// dépend d'aucun contrat ouvert, et l'ensemble des erreurs reste énumérable.
type Error struct {
	// Code identifie l'issue. C'est lui que les surfaces traduisent.
	Code ErrorCode
	// Message est destiné à l'utilisateur : aucun détail technique, aucune
	// donnée sensible.
	Message string
	// Field nomme le champ fautif, pour une erreur de validation.
	Field string
	// cause porte le détail technique. Journalisée, JAMAIS retournée à
	// l'appelant : une erreur SQL renvoyée au client est une fuite de structure.
	cause error
}

// Ack signale un effet accompli qui n'a aucune valeur à rendre.
//
// Existe pour que `ports/` ne contienne AUCUNE déclaration de structure, pas même
// le `struct{}` anonyme de `Result[struct{}, Error]` — la règle d'architecture le
// refuse, et elle a raison : un port doit se lire comme une signature, pas comme un
// type. Le gain est aussi à l'appel, où `domain.Ack{}` se lit mieux que
// `struct{}{}`.
type Ack struct{}

// NewError construit une erreur métier.
func NewError(code ErrorCode, message string) Error {
	return Error{Code: code, Message: message}
}

// WithField précise le champ fautif et retourne une nouvelle erreur.
func (e Error) WithField(field string) Error {
	e.Field = field
	return e
}

// WithCause attache un détail technique et retourne une nouvelle erreur.
func (e Error) WithCause(cause error) Error {
	e.cause = cause
	return e
}

// Cause expose le détail technique, pour la journalisation uniquement.
func (e Error) Cause() error { return e.cause }

// Error rend Error compatible avec l'interface error, ce qui simplifie les
// frontières. Le message technique n'y apparaît pas.
func (e Error) Error() string {
	if e.Field != "" {
		return string(e.Code) + " (" + e.Field + "): " + e.Message
	}
	return string(e.Code) + ": " + e.Message
}
