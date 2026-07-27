// Package memory implémente la persistance de l'inscription en mémoire.
//
// # Pourquoi ce pilote existe
//
// Il n'est pas un bouchon de test : c'est le pilote PAR DÉFAUT du module, et
// c'est lui qui rend vraie la promesse du socle — `hexa new` puis `go run`
// démarre sans base, sans Docker, sans rien. Un module métier dont le seul
// pilote exigerait PostgreSQL casserait cette promesse au premier module écrit,
// c'est-à-dire au moment exact où l'on cherche à l'éprouver.
//
// # GARANTIES
//
//   - **Unicité de l'adresse** : la carte est indexée par adresse normalisée, et
//     `SaveUser` refuse un doublon avec `CodeEmailAlreadyExists` — le même code
//     que le pilote SQL rendra sur une violation de contrainte. C'est ce qui rend
//     les deux pilotes réellement substituables (ADR 003).
//   - **Sûr en concurrence** : toutes les lectures et écritures passent par un
//     RWMutex.
//   - **Aucune mutation de l'entrée** : `domain.User` est une valeur, elle est
//     copiée à l'entrée comme à la sortie.
//
// # NON-GARANTIES
//
//   - **Aucune durabilité.** Tout disparaît à l'arrêt du processus. C'est le
//     comportement attendu d'un pilote mémoire, et il doit être choisi en
//     connaissance de cause — jamais subi.
//   - **Aucun partage entre répliques.** Deux instances ont deux magasins, donc
//     deux vérités. L'unicité de l'adresse n'est garantie QUE dans un processus.
//   - **Aucune transaction.** `RunInTx` du module se réduit à un appel direct :
//     un échec après l'écriture ne l'annule pas. C'est la NON-garantie la plus
//     coûteuse, et la raison pour laquelle ce pilote ne convient pas à la
//     production.
//   - **La mémoire croît sans fin.** Aucune purge, aucune borne.
package memory

import (
	"context"
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Store retient les utilisateurs inscrits, indexés par adresse normalisée.
//
// L'index est l'adresse et non l'identifiant : toutes les opérations du module
// interrogent par adresse. Indexer par identifiant obligerait à parcourir la
// carte à chaque vérification d'unicité — et surtout, rendrait l'unicité de
// l'adresse dépendante d'une discipline d'appel plutôt que de la structure.
type Store struct {
	mu      sync.RWMutex
	byEmail map[domain.Email]domain.User
}

// New construit un magasin vide.
func New() *Store {
	return &Store{byEmail: make(map[domain.Email]domain.User)}
}

// Save implémente ports.SaveUser.
//
// Le refus du doublon est ici et non dans le cas d'usage, délibérément. Le cas
// d'usage vérifie déjà la disponibilité, mais entre sa vérification et son
// écriture il existe une fenêtre : deux inscriptions simultanées sur la même
// adresse la franchissent toutes les deux. Seul le magasin, qui détient le
// verrou, peut trancher — exactement comme la contrainte d'unicité SQL tranche
// pour le pilote postgres.
func (s *Store) Save(_ context.Context, user domain.User) result.Result[domain.User, domain.Error] {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return result.Err[domain.User, domain.Error](domain.NewError(
			domain.CodeEmailAlreadyExists,
			"cette adresse de courriel est déjà enregistrée",
		).WithField("email"))
	}

	s.byEmail[user.Email] = user
	return result.Ok[domain.User, domain.Error](user)
}

// IsTaken implémente ports.EmailIsTaken.
//
// L'absence n'est PAS une erreur : elle vaut `false`. C'est le contrat du port.
func (s *Store) IsTaken(_ context.Context, email domain.Email) result.Result[bool, domain.Error] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.byEmail[email]
	return result.Ok[bool, domain.Error](exists)
}

// FindByEmail implémente ports.FindUserByEmail.
//
// Une absence retourne `None`, jamais une erreur : l'Option rend l'absence
// explicite dans le type, donc impossible à confondre avec une panne.
func (s *Store) FindByEmail(
	_ context.Context,
	email domain.Email,
) result.Result[fp.Option[domain.User], domain.Error] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byEmail[email]
	if !exists {
		return result.Ok[fp.Option[domain.User], domain.Error](fp.None[domain.User]())
	}
	return result.Ok[fp.Option[domain.User], domain.Error](fp.Some(user))
}

// Count rend le nombre d'utilisateurs retenus.
//
// Exposé pour l'exploitation et les sondes, pas pour les tests : ceux-ci passent
// par les ports, comme n'importe quel appelant.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byEmail)
}
