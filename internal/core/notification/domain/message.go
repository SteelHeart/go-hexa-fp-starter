package domain

import (
	"fmt"
	"strings"
)

// Bornes d'un message.
//
// Explicites plutôt qu'absentes : sans elles, un corps de plusieurs mégaoctets
// traverserait le domaine jusqu'au fournisseur, où il échouerait avec un message
// de bibliothèque tierce au lieu d'un message métier.
const (
	subjectMaxLen = 300
	bodyMaxLen    = 1 << 20
)

// Channel est le moyen d'acheminement.
//
// Un type nommé plutôt qu'une chaîne : c'est ce qui empêche de passer un sujet
// là où un canal est attendu, confusion que le compilateur ne verrait jamais
// entre deux `string`.
type Channel string

// Canaux SERVIS par ce module.
//
// La liste dit ce qui EXISTE, pas ce qui est prévu (ADR 014). `sms` et `push`
// sont décrits dans documentation/technique/modules-noyau.md et ne sont pas
// livrés : les déclarer ici les rendrait sélectionnables, et un message partirait
// vers un canal que rien n'implémente.
const (
	// ChannelEmail est le seul canal livré.
	ChannelEmail Channel = "email"
)

// Message est ce qu'on demande d'acheminer.
//
// Le CONTENU est fourni rendu, pas gabarité : ce module ne connaît aucun moteur
// de gabarits, donc il n'en impose aucun. Le rendu appartient à l'appelant, qui
// sait dans quelle langue et avec quelles données écrire.
type Message struct {
	Channel Channel
	To      Recipient
	Subject string
	Body    string
}

// NewMessage valide et normalise un message. Seul chemin de construction utile.
//
// # Pourquoi le canal est validé ICI et non au pilote
//
// Un pilote qui refuserait le canal le ferait après que le message a traversé
// le cas d'usage, donc après qu'un décorateur a pu le journaliser ou le compter
// comme envoyé. Le refus le plus utile est le plus précoce.
func NewMessage(channel Channel, to Recipient, subject, body string) (Message, error) {
	if channel != ChannelEmail {
		return Message{}, fmt.Errorf("%w: %q — seul %q est livré", ErrUnknownChannel, channel, ChannelEmail)
	}
	if to.IsZero() {
		return Message{}, fmt.Errorf("%w: le destinataire est obligatoire", ErrIncomplete)
	}

	trimmedSubject := strings.TrimSpace(subject)
	switch {
	case trimmedSubject == "":
		return Message{}, fmt.Errorf("%w: le sujet est obligatoire", ErrIncomplete)
	case len(trimmedSubject) > subjectMaxLen:
		return Message{}, fmt.Errorf("%w: le sujet dépasse %d caractères", ErrIncomplete, subjectMaxLen)
	case strings.TrimSpace(body) == "":
		return Message{}, fmt.Errorf("%w: le corps est obligatoire", ErrIncomplete)
	case len(body) > bodyMaxLen:
		return Message{}, fmt.Errorf("%w: le corps dépasse %d octets", ErrIncomplete, bodyMaxLen)
	}

	return Message{Channel: channel, To: to, Subject: trimmedSubject, Body: body}, nil
}

// String rend une forme SANS le corps ni l'adresse en clair.
//
// # Ce que le masquage empêche
//
// Un corps de notification transporte régulièrement un secret : lien de
// confirmation, jeton de réinitialisation, code à usage unique. Ce sont des
// identifiants au porteur — qui les lit peut s'en servir. Les journaliser
// transforme une fuite de journaux en prise de contrôle de comptes.
//
// `Stringer` ET `GoStringer` sont implémentés, pas par redondance : `%v` passe
// par le premier, `%#v` par le second, et couvrir l'un laisse l'autre fuiter.
// Il a fallu un test pour le découvrir sur `auth.Credential`.
func (m Message) String() string {
	return fmt.Sprintf(
		"Message{channel: %s, to: %s, subject: %q, body: *** (%d octets)}",
		m.Channel, m.To.Masked(), m.Subject, len(m.Body))
}

// GoString masque le corps sous `%#v` aussi.
func (m Message) GoString() string { return m.String() }
