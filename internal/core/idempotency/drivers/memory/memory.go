// Package memory implémente l'idempotence en mémoire.
//
// # NON-GARANTIES — à lire avant de l'utiliser
//
//   - **Aucun partage entre répliques.** Deux instances derrière un répartiteur
//     ne se voient pas : la même requête rejouée sur l'autre instance s'exécutera
//     deux fois. C'est la garantie même du module qui disparaît.
//   - **Aucune durabilité.** Tout est perdu au redémarrage : un rejeu arrivé
//     juste après un déploiement s'exécutera une seconde fois.
//   - **Aucune expiration passive.** Les clés ne disparaissent qu'au passage de
//     `Purge` : sans ordonnanceur, la carte croît sans borne.
//
// Convient en développement, en test unitaire, pour une CLI et pour tout binaire
// mono-instance. Ne convient PAS dès qu'il y a deux répliques.
//
// Ce n'est pas un bouchon pour autant : l'exclusivité de `Reserve` est réellement
// tenue dans le processus, et le pilote passe la même suite de conformité que le
// pilote `postgres`.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// entry est une réservation vivante ou achevée.
type entry struct {
	fingerprint string
	status      domain.Status
	response    []byte
	expiresAt   time.Time
}

// Store est le magasin en mémoire.
type Store struct {
	mu      sync.Mutex
	entries map[domain.Key]*entry
	ttl     time.Duration
	now     func() time.Time
}

// New construit le magasin.
//
// `now` est injecté : un test d'expiration ne doit pas attendre.
func New(ttl time.Duration, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{entries: make(map[domain.Key]*entry), ttl: ttl, now: now}
}

// Reserve implémente ports.Reserve.
//
// L'exclusivité vient du mutex tenu pendant toute la décision : la lecture de la
// réservation existante et l'écriture de la nouvelle sont indivisibles. Relâcher
// le verrou entre les deux laisserait deux appels concurrents obtenir la clé.
func (s *Store) Reserve(ctx context.Context, req domain.Request) (domain.Reservation, error) {
	if !req.IsComplete() {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrIncomplete, req.Key)
	}
	if err := ctx.Err(); err != nil {
		return domain.Reservation{}, fmt.Errorf("réservation annulée: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if held, found := s.entries[req.Key]; found && held.expiresAt.After(now) {
		return decide(*held, req)
	}
	s.entries[req.Key] = &entry{
		fingerprint: req.Fingerprint,
		status:      domain.StatusInFlight,
		expiresAt:   now.Add(s.ttl),
	}
	return domain.Reservation{}, nil
}

// decide tranche le sort d'une réservation vivante — issues 3, 4 et 5 du
// contrat de ports.Reserve.
func decide(held entry, req domain.Request) (domain.Reservation, error) {
	if held.fingerprint != req.Fingerprint {
		return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrConflict, req.Key)
	}
	if held.status == domain.StatusDone {
		return domain.Reservation{Replayed: true, Response: clone(held.response)}, nil
	}
	return domain.Reservation{}, fmt.Errorf("%w: clé=%q", domain.ErrInFlight, req.Key)
}

// Complete implémente ports.Complete.
func (s *Store) Complete(_ context.Context, key domain.Key, response []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	held, found := s.entries[key]
	if !found || !held.expiresAt.After(s.now()) {
		return fmt.Errorf("%w: clé=%q", domain.ErrNotReserved, key)
	}
	held.status = domain.StatusDone
	held.response = clone(response)
	return nil
}

// Release implémente ports.Release.
//
// Une clé achevée n'est jamais libérée : ce serait rouvrir la porte au rejeu.
func (s *Store) Release(_ context.Context, key domain.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if held, found := s.entries[key]; found && held.status == domain.StatusInFlight {
		delete(s.entries, key)
	}
	return nil
}

// Purge implémente ports.Purge.
func (s *Store) Purge(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	var removed int64
	for key, held := range s.entries {
		if !held.expiresAt.After(now) {
			delete(s.entries, key)
			removed++
		}
	}
	return removed, nil
}

// clone recopie la charge.
//
// Sans cette copie, l'appelant garderait une référence sur la mémoire du magasin
// et pourrait altérer une réponse mémorisée — un rejeu rendrait alors autre chose
// que le premier appel, silencieusement.
func clone(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
