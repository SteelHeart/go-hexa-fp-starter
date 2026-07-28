// Package ports déclare les contrats de la notification.
//
// Ce paquet ne contient QUE des déclarations de types : ni struct, ni fonction,
// ni interface. Un port est un type fonction — la plus petite interface
// possible, donc rien à ségréguer (ADR 003).
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
)

// ─── Port primaire : ce que le monde extérieur peut demander ─────────────────

// Send achemine un message.
//
// # Ce que ce port ne promet PAS
//
// Ni la réception, ni la lecture, ni même la remise finale. Il promet que le
// message a été ACCEPTÉ par le fournisseur. Un courriel accepté peut encore être
// rejeté par le serveur destinataire une minute plus tard, et aucun code
// synchrone ne peut le savoir.
//
// Promettre davantage ferait écrire des appelants qui traitent le succès comme
// une preuve de réception — et cette croyance se paie au premier support client
// qui affirme n'avoir rien reçu.
//
// Erreurs : `domain.ErrIncomplete` pour un message mal formé,
// `domain.ErrUnknownChannel` pour un canal non servi, `domain.ErrUndeliverable`
// quand le fournisseur refuse. La distinction décide du réessai — on rejoue une
// panne, jamais une adresse invalide.
type Send = func(ctx context.Context, message domain.Message) error

// ─── Port secondaire : ce dont le cœur a besoin du monde ─────────────────────

// Deliver remet le message au fournisseur.
//
// Reçoit un message DÉJÀ VALIDÉ : le pilote n'a pas à revalider, et surtout ne
// doit pas. Deux validations pour une même règle, c'est une de trop — elles
// divergeront, et c'est la mauvaise qui décidera.
//
// Contrat d'erreur : `domain.ErrUndeliverable`, enveloppée avec la cause.
type Deliver = func(ctx context.Context, message domain.Message) error
