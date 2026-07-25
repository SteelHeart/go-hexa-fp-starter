// Package ports déclare les contrats de l'outbox.
//
// Ce paquet ne contient QUE des déclarations de types : ni struct, ni fonction,
// ni interface (vérifié par arch-go.yml).
//
// # Pourquoi `error` et non `Result[T, domain.Error]`
//
// Un module NOYAU est technique : l'outbox n'a pas de taxonomie d'erreur métier
// à exposer. `rules/programmation-fonctionnelle.md` §4 réserve `Result` au cœur
// métier et admet `error` nu dans le technique. Un module MÉTIER, lui, retourne
// toujours `Result[T, domain.Error]`.
//
// Cette frontière est nette et vérifiable : `internal/core/**` utilise `error`,
// `internal/modules/**` utilise `Result`.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// Enqueue persiste une intention de publication.
//
// À appeler DANS la transaction métier : c'est tout l'intérêt du patron. Le
// pilote `postgres` lit le querier du contexte, donc l'atomicité est acquise
// sans que l'appelant s'en occupe.
//
// Contrat d'erreur : toute erreur est technique. Le pilote `memory` ne peut
// échouer que sur un contexte annulé.
type Enqueue = func(ctx context.Context, msg domain.NewMessage) (domain.MessageID, error)

// Claim réserve un lot de messages dus et les verrouille.
//
// Contrat : deux appels concurrents ne retournent JAMAIS le même message. Le
// pilote `postgres` l'obtient par `FOR UPDATE SKIP LOCKED`, le pilote `memory`
// par un verrou local. Un pilote qui ne peut pas le garantir n'est pas conforme.
type Claim = func(ctx context.Context, limit int) ([]domain.Message, error)

// MarkDone marque un message publié avec succès.
type MarkDone = func(ctx context.Context, id domain.MessageID) error

// MarkFailed enregistre un échec et programme le prochain essai.
//
// Le calcul du recul n'est PAS ici : il est dans domain.NextAttempt, donc pur et
// testable. Le pilote ne fait qu'écrire.
type MarkFailed = func(ctx context.Context, attempt domain.FailedAttempt) error

// PendingCount compte les messages en attente.
//
// C'est la métrique la plus importante du système : elle croît quand le worker
// est mort, et c'est le seul symptôme visible d'une chaîne asynchrone en panne.
type PendingCount = func(ctx context.Context) (int64, error)

// Handler traite un message.
//
// Il DOIT être idempotent : le dépilage est « au moins une fois », donc tout
// message sera rejoué au moins une fois dans la vie du système.
type Handler = func(ctx context.Context, msg domain.Message) error

// Report rend compte du traitement d'un message.
//
// L'orchestration ne journalise pas : elle rend compte. C'est ce qui la garde
// pure — `rules/README.md` interdit tout logger dans `application/` — et ce qui
// permet à un test de vérifier une politique de dépilage en lisant des valeurs.
//
// Ne retourne rien : un compte rendu qui échouerait ne doit jamais faire échouer
// le dépilage dont il rend compte.
type Report = func(ctx context.Context, outcome domain.Outcome)

// Now rend l'instant courant.
//
// L'orchestration ne lit pas l'horloge du système : elle reçoit ce port. Sans
// cela, la politique de recul serait intestable sans attendre réellement.
type Now = func() time.Time
