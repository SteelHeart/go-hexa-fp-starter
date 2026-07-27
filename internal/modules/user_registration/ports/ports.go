// Package ports déclare les contrats de la feature.
//
// Ce paquet ne contient QUE des déclarations de types : ni struct, ni fonction,
// ni interface (vérifié par arch-go.yml). Un port est un type fonction —
// la plus petite interface possible, donc rien à ségréguer
// (documentation/adr/003).
//
// Chaque port secondaire porte son CONTRAT D'ERREUR en commentaire : c'est la
// forme opérationnelle de la substituabilité, et il est vérifié par la suite de
// conformité partagée entre implémentations.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// ─── Ports primaires : ce que le monde extérieur peut demander ───────────────

// RegisterUser inscrit un nouvel utilisateur.
//
// Toutes les surfaces appellent CETTE fonction : HTTP, CLI, seeders. Ajouter une
// surface n'en change pas la signature (documentation/adr/005).
//
// Erreurs : CodeInvalidEmail · CodeWeakPassword · CodeEmailAlreadyExists ·
// CodeUnavailable · CodeInternal.
type RegisterUser = func(
	ctx context.Context,
	cmd domain.RegistrationCommand,
) result.Result[domain.User, domain.Error]

// CheckEmailAvailability indique si une adresse est encore libre.
//
// Port de lecture : c'est lui qui porte le décorateur de cache, un cache sur une
// écriture n'ayant aucun sens.
//
// Erreurs : CodeInvalidEmail · CodeUnavailable · CodeInternal.
type CheckEmailAvailability = func(
	ctx context.Context,
	rawEmail string,
) result.Result[bool, domain.Error]

// ─── Ports secondaires : ce dont le cœur a besoin du monde ───────────────────

// SaveUser persiste un utilisateur nouvellement inscrit.
//
// Contrat d'erreur — toute implémentation DOIT respecter :
//   - CodeEmailAlreadyExists si la contrainte d'unicité d'adresse est violée
//   - CodeUnavailable        si le stockage est injoignable ou la requête annulée
//   - CodeInternal           pour tout le reste, avec la cause attachée
//
// Aucune erreur de pilote ne doit remonter telle quelle.
type SaveUser = func(
	ctx context.Context,
	user domain.User,
) result.Result[domain.User, domain.Error]

// EmailIsTaken indique si une adresse est déjà enregistrée.
//
// Contrat d'erreur : CodeUnavailable · CodeInternal. L'absence de ligne n'est
// PAS une erreur — elle vaut `false`.
type EmailIsTaken = func(
	ctx context.Context,
	email domain.Email,
) result.Result[bool, domain.Error]

// FindUserByEmail cherche un utilisateur par son adresse.
//
// Contrat d'erreur : CodeUnavailable · CodeInternal. Une absence retourne
// `None`, jamais une erreur : l'Option rend l'absence explicite dans le type.
type FindUserByEmail = func(
	ctx context.Context,
	email domain.Email,
) result.Result[fp.Option[domain.User], domain.Error]

// HashPassword produit le condensé d'un mot de passe.
//
// Contrat d'erreur : CodeInternal uniquement. Un échec de hachage est un défaut
// technique, jamais une erreur utilisateur.
type HashPassword = func(
	password domain.RawPassword,
) result.Result[domain.PasswordHash, domain.Error]

// PublishEvent enregistre un événement à publier.
//
// L'implémentation écrit dans l'outbox, DANS la transaction courante. Le cœur ne
// connaît aucun broker (documentation/adr/006).
//
// Contrat d'erreur : CodeUnavailable · CodeInternal.
type PublishEvent = func(
	ctx context.Context,
	eventType string,
	aggregateID string,
	payload any,
) result.Result[domain.Ack, domain.Error]

// SendWelcomeEmail envoie le courriel de bienvenue.
//
// Appelé par le consommateur d'événements, jamais par le cas d'usage
// d'inscription : l'envoi ne doit pas faire échouer l'inscription.
//
// Contrat d'erreur : CodeUnavailable · CodeInternal.
type SendWelcomeEmail = func(
	ctx context.Context,
	user domain.User,
) result.Result[domain.Ack, domain.Error]

// ─── Ports d'effets purs : l'horloge et l'aléa ───────────────────────────────

// Now retourne l'instant courant.
//
// C'est un port PRÉCISÉMENT pour que les tests soient déterministes : un test
// qui lit l'horloge réelle est un test qui échouera un jour, sans raison.
type Now = func() time.Time

// GenerateID produit un identifiant d'utilisateur.
type GenerateID = func() domain.UserID

// ─── Unité de travail ────────────────────────────────────────────────────────

// RunInTx exécute une fonction dans une transaction.
//
// Le rollback est déclenché par un Result en Err : une inscription refusée ne
// laisse donc aucun événement dans l'outbox.
type RunInTx = func(
	ctx context.Context,
	fn func(context.Context) result.Result[domain.User, domain.Error],
) result.Result[domain.User, domain.Error]
