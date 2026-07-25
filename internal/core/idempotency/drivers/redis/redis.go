// Package redis implémente l'idempotence sur Redis.
//
// # GARANTIES
//
//   - **Exclusivité entre répliques** : `SET NX` est atomique côté serveur. Une
//     seule des requêtes concurrentes obtient la clé.
//   - **Expiration passive** : Redis efface les clés tout seul, donc aucune purge
//     n'est nécessaire et le magasin ne croît pas sans borne.
//
// # NON-GARANTIES
//
//   - **Aucune atomicité avec la base métier.** Redis et la base sont deux
//     magasins : un `Complete` peut réussir alors que la transaction métier a
//     échoué. L'appelant doit appeler `Release` sur son chemin d'échec.
//   - **`Complete` n'est pas atomique** : il lit l'empreinte puis réécrit
//     (`SET XX`). Une expiration glissée entre les deux fait échouer la
//     mémorisation, jamais l'opération métier.
//   - **Aucune durabilité forte.** Un Redis sans persistance, ou vidé, perd les
//     mémorisations : les rejeux en vol s'exécuteront une seconde fois.
//   - **Les réponses vivent en mémoire.** Mémoriser de grosses charges pèse sur
//     l'instance ; réserver ce pilote à des réponses courtes.
//
// # État
//
// Écrit, JAMAIS exécuté contre un Redis réel. Ne pas le présenter comme éprouvé.
package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// envelope est ce qui est stocké sous la clé.
//
// Un seul enregistrement porte l'empreinte, le statut et la réponse : la
// décision de `Reserve` se prend donc sur UNE lecture, sans état partiel possible
// entre plusieurs clés Redis.
type envelope struct {
	Fingerprint string        `json:"fingerprint"`
	Status      domain.Status `json:"status"`
	Response    []byte        `json:"response,omitempty"`
}

// Store implémente l'idempotence sur Redis.
type Store struct {
	client    *goredis.Client
	namespace string
	ttl       time.Duration
}

// New construit le magasin.
//
// `namespace` préfixe toutes les clés : deux déploiements partageant une instance
// Redis ne doivent pas se voler leurs réservations.
func New(client *goredis.Client, namespace string, ttl time.Duration) *Store {
	return &Store{client: client, namespace: namespace, ttl: ttl}
}

// key qualifie la clé applicative.
func (s *Store) key(k domain.Key) string { return s.namespace + ":" + k.String() }

// Reserve implémente ports.Reserve.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrIncomplete, req.Key)
	}

	raw, err := json.Marshal(envelope{Fingerprint: req.Fingerprint, Status: domain.StatusInFlight})
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("sérialisation de la réservation: %w", err)
	}

	// SET NX pose la clé ET son expiration en une seule commande : un plantage
	// entre les deux laisserait sinon une réservation éternelle.
	won, err := s.client.SetNX(ctx, s.key(req.Key), raw, s.ttl).Result()
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("réservation de la clé d'idempotence: %w", err)
	}
	if won {
		return domain.Reservation{}, nil
	}
	return s.inspect(ctx, req)
}

// inspect lit la réservation qui nous a devancés et tranche.
func (s *Store) inspect(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	held, err := s.read(ctx, req.Key)
	if err != nil {
		// Clé disparue entre le SET NX refusé et cette lecture, ou panne : on
		// refuse. Faire réessayer un client est bénin, exécuter deux fois non.
		if errors.Is(err, domain.ErrNotReserved) {
			return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrInFlight, req.Key)
		}
		return domain.Reservation{}, err
	}
	if held.Fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrConflict, req.Key)
	}
	if held.Status == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: held.Response}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrInFlight, req.Key)
}

// read charge l'enregistrement. Une absence rend domain.ErrNotReserved.
func (s *Store) read(ctx context.Context, key domain.Key) (envelope, error) {
	raw, err := s.client.Get(ctx, s.key(key)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return envelope{}, fmt.Errorf("%w: clé=%q", domain.ErrNotReserved, key)
	}
	if err != nil {
		return envelope{}, fmt.Errorf("lecture de la clé d'idempotence: %w", err)
	}
	var held envelope
	if err := json.Unmarshal(raw, &held); err != nil {
		return envelope{}, fmt.Errorf("enregistrement d'idempotence illisible: %w", err)
	}
	return held, nil
}

// Complete implémente ports.Complete.
func (s *Store) Complete(ctx context.Context, key domain.Key, response []byte) error {
	held, err := s.read(ctx, key)
	if err != nil {
		return err
	}
	if held.Status != domain.StatusInFlight {
		return fmt.Errorf("%w: clé=%q déjà résolue", domain.ErrNotReserved, key)
	}

	raw, err := json.Marshal(envelope{
		Fingerprint: held.Fingerprint,
		Status:      domain.StatusDone,
		Response:    response,
	})
	if err != nil {
		return fmt.Errorf("sérialisation de la réponse idempotente: %w", err)
	}

	// KeepTTL : la fenêtre de rejeu court depuis la réservation. Sans lui, chaque
	// mémorisation repousserait l'expiration et les clés s'accumuleraient.
	kept, err := s.client.SetXX(ctx, s.key(key), raw, goredis.KeepTTL).Result()
	if err != nil {
		return fmt.Errorf("mémorisation de la réponse idempotente: %w", err)
	}
	if !kept {
		return fmt.Errorf("%w: clé=%q expirée pendant l'opération", domain.ErrNotReserved, key)
	}
	return nil
}

// Release implémente ports.Release.
//
// Ne supprime qu'une réservation en cours : effacer une clé achevée rouvrirait la
// porte au rejeu.
func (s *Store) Release(ctx context.Context, key domain.Key) error {
	held, err := s.read(ctx, key)
	if errors.Is(err, domain.ErrNotReserved) {
		return nil
	}
	if err != nil {
		return err
	}
	if held.Status != domain.StatusInFlight {
		return nil
	}
	if err := s.client.Del(ctx, s.key(key)).Err(); err != nil {
		return fmt.Errorf("libération de la clé d'idempotence: %w", err)
	}
	return nil
}

// Purge implémente ports.Purge.
//
// Sans effet par CONCEPTION, non par omission : Redis expire les clés lui-même.
// Retourner 0 sans erreur permet à l'ordonnanceur d'appeler le port quel que soit
// le pilote branché.
func (s *Store) Purge(_ context.Context) (int64, error) { return 0, nil }
