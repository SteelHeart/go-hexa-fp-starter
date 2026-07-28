// Package application compose les cas d'usage de l'authentification.
//
// Il ne journalise pas et ne lit pas l'horloge : il rend compte par ses retours
// et reçoit son instant. C'est ce qui le garde testable sans analyser de
// journaux, et déterministe sans attendre.
//
// # Carte des fichiers
//
//	register.go       créer une identité et son secret
//	authenticate.go   échanger un secret contre une session
//	verify.go         résoudre un jeton en identité
//	authorize.go      vérifier une permission — contre l'état PERSISTÉ
package application

import (
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// Deps porte les ports dont les cas d'usage ont besoin.
//
// Tous sont des types fonction : en test, chacun est une closure de trois lignes,
// et aucune bibliothèque de simulacre n'est nécessaire — donc aucune n'est
// autorisée (rules/dependances.md).
type Deps struct {
	SaveIdentity  ports.SaveIdentity
	FindBySubject ports.FindBySubject
	FindIdentity  ports.FindIdentity
	SaveSession   ports.SaveSession
	FindSession   ports.FindSession
	DeleteSession ports.DeleteSession
	Grants        ports.Grants
	SaveRole      ports.SaveRole
	BindRoles     ports.BindRoles

	HashSecret   ports.HashSecret
	VerifySecret ports.VerifySecret

	Now           ports.Now
	NewToken      ports.NewToken
	NewIdentityID ports.NewIdentityID

	// SessionTTL borne la durée d'une session.
	//
	// Portée par les dépendances et non par le domaine : c'est un réglage
	// d'exploitation, pas une règle métier. Une valeur nulle ou négative fait
	// refuser la construction de la session — une session éternelle ne se décide
	// pas par omission.
	SessionTTL time.Duration
}
