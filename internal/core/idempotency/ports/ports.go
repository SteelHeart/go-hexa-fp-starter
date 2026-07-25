// Package ports déclare les contrats de l'idempotence.
//
// Ce paquet ne contient QUE des déclarations de types : ni struct, ni fonction,
// ni interface.
//
// # Pourquoi `error` et non `Result[T, domain.Error]`
//
// Un module NOYAU est technique : l'idempotence n'a pas de taxonomie d'erreur
// métier à exposer. La frontière est nette et vérifiable — `internal/core/**`
// utilise `error`, `internal/modules/**` utilise `Result`.
//
// # Le protocole, en trois appels
//
//	res, err := reserve(ctx, domain.Request{Key: k, Fingerprint: fp})
//	switch {
//	case errors.Is(err, domain.ErrInFlight): // 409 : réessayer, ne PAS exécuter
//	case errors.Is(err, domain.ErrConflict): // 422 : le client a fauté
//	case err != nil:                        // panne technique
//	case res.Replayed:                      // retourner res.Response telle quelle
//	default:                                // exécuter, puis complete() ou release()
//	}
//
// Oublier `release()` sur échec rend l'opération impossible jusqu'à expiration de
// la clé : le remède serait pire que le mal. `defer` est le bon réflexe.
package ports

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// Reserve tente d'obtenir l'exclusivité pour une clé.
//
// # Contrat de conformité — tout pilote doit rendre ces cinq issues
//
//  1. Requête incomplète → `domain.ErrIncomplete`. Jamais de clé vide acceptée.
//  2. Clé libre ou réservation expirée → `Reservation{}` et l'appelant exécute.
//  3. Clé connue, empreinte différente → `domain.ErrConflict`.
//  4. Clé connue, opération achevée → `Reservation{Replayed: true, Response: …}`.
//  5. Clé connue, opération en cours → `domain.ErrInFlight`.
//
// L'issue 5 exige de l'atomicité : deux appels concurrents sur une clé libre ne
// peuvent JAMAIS obtenir tous les deux l'issue 2. Un pilote incapable de le
// garantir n'est pas conforme — c'est la seule promesse du module.
//
// En cas de doute, un pilote refuse (`ErrInFlight`) plutôt que d'autoriser :
// faire réessayer un client est bénin, exécuter deux fois ne l'est pas.
type Reserve = func(ctx context.Context, req domain.Request) (domain.Reservation, error)

// Complete mémorise la réponse pour les rejeux ultérieurs.
//
// Retourne `domain.ErrNotReserved` si la réservation n'existe plus. L'opération
// métier, elle, a réussi : l'appelant journalise et poursuit.
type Complete = func(ctx context.Context, key domain.Key, response []byte) error

// Release libère une clé après un échec, pour que l'appelant puisse réessayer.
//
// Ne libère JAMAIS une clé achevée : ce serait rouvrir la porte au rejeu que le
// module existe pour fermer.
type Release = func(ctx context.Context, key domain.Key) error

// Purge supprime les clés expirées et retourne leur nombre.
//
// À appeler périodiquement par l'ordonnanceur. Un pilote dont le magasin expire
// tout seul retourne 0 sans erreur.
type Purge = func(ctx context.Context) (int64, error)
