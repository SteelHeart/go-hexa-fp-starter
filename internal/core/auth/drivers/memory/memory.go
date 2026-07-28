// Package memory implémente le magasin d'authentification en mémoire.
//
// # Pourquoi ce pilote existe
//
// Il n'est pas un bouchon de test : c'est le pilote PAR DÉFAUT, et c'est lui qui
// rend vraie la promesse du socle — `hexa new` puis `go run` démarre sans base,
// sans Docker, **avec l'authentification active**. Un module `auth` dont le seul
// pilote exigerait PostgreSQL casserait cette promesse au moment exact où l'on
// cherche à la démontrer.
//
// # Carte des fichiers
//
//	memory.go       le magasin lui-même, et ses garanties
//	identities.go   identités et condensés
//	sessions.go     jetons émis, révocation, purge
//	roles.go        rôles, affectation, et la question « détient-il ce droit ? »
//
// # GARANTIES
//
//   - **Unicité du sujet** : indexé par sujet normalisé, `SaveIdentity` refuse un
//     doublon avec `ErrSubjectTaken` — le même code qu'un pilote SQL rendra sur
//     une violation de contrainte. C'est ce qui rend deux pilotes réellement
//     substituables.
//   - **Révocation immédiate** : une session supprimée cesse de valoir au prochain
//     appel, un rôle retiré cesse d'accorder au prochain appel, et un compte fermé
//     cesse de valoir partout au prochain appel. C'est la décision 1 de
//     l'ADR 017, tenue par le magasin.
//   - **Sûr en concurrence** : toutes les lectures et écritures passent par un
//     RWMutex.
//
// # NON-GARANTIES
//
//   - **Aucune durabilité.** Tout disparaît à l'arrêt : redémarrer déconnecte tout
//     le monde et EFFACE LES COMPTES. C'est acceptable en développement et
//     inacceptable ailleurs.
//   - **Aucun partage entre répliques.** Deux instances ont deux magasins : une
//     session émise par l'une est inconnue de l'autre. Derrière un répartiteur de
//     charge, une requête sur deux échouerait en 401.
//   - **Aucune purge automatique.** Les sessions expirées restent en mémoire
//     jusqu'à un appel explicite à `PurgeExpired` ; elles ne valent plus rien,
//     mais elles occupent.
package memory

import (
	"sync"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// Store retient identités, secrets, rôles et sessions.
//
// Un SEUL verrou pour les quatre cartes, délibérément. Elles sont lues ensemble —
// `Grants` traverse les créances ET les rôles — et des verrous séparés ne
// gagneraient rien qu'un ordre de prise à respecter, donc un interblocage à
// découvrir en production.
type Store struct {
	mu          sync.RWMutex
	bySubject   map[string]domain.IdentityID
	credentials map[domain.IdentityID]domain.Credential
	roles       map[string]domain.Role
	sessions    map[string]domain.Session
}

// New construit un magasin vide.
func New() *Store {
	return &Store{
		bySubject:   make(map[string]domain.IdentityID),
		credentials: make(map[domain.IdentityID]domain.Credential),
		roles:       make(map[string]domain.Role),
		sessions:    make(map[string]domain.Session),
	}
}
