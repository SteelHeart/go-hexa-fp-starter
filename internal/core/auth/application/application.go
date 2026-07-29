// Package application composes the authentication use cases.
//
// It does not log and does not read the clock: it reports through its return
// values and receives its instant. That is what keeps it testable without
// parsing logs, and deterministic without waiting.
//
// # File map
//
//	register.go       create an identity and its secret
//	authenticate.go   exchange a secret for a session
//	verify.go         resolve a token into an identity, and revoke
//	authorize.go      check a permission — against the PERSISTED state
//	roles.go          define a role, assign it
//	identities.go     close an account, reopen it
package application

import (
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// Deps carries the ports the use cases need.
//
// All of them are function types: in a test, each one is a three-line closure,
// and no mocking library is necessary — hence none is allowed
// (rules/dependances.md).
type Deps struct {
	SaveIdentity   ports.SaveIdentity
	FindBySubject  ports.FindBySubject
	FindIdentity   ports.FindIdentity
	UpdateIdentity ports.UpdateIdentity
	SaveSession    ports.SaveSession
	FindSession    ports.FindSession
	DeleteSession  ports.DeleteSession
	Grants         ports.Grants
	SaveRole       ports.SaveRole
	BindRoles      ports.BindRoles

	HashSecret   ports.HashSecret
	VerifySecret ports.VerifySecret

	Now           ports.Now
	NewToken      ports.NewToken
	NewIdentityID ports.NewIdentityID

	// SessionTTL bounds the lifetime of a session.
	//
	// Carried by the dependencies and not by the domain: it is an operational
	// setting, not a business rule. A zero or negative value makes the session
	// construction refuse — an eternal session is not decided by omission.
	SessionTTL time.Duration
}
