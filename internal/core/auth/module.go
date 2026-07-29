// Package auth is the composition root of the authentication module.
//
// This is the ONLY file in the module that knows about drivers (ADR 012). Use
// cases only ever see function types: switching driver does not touch a single
// line of `application/` or of `domain/`.
//
// # This module returns an `error`, not a `Result`
//
// `internal/core/**` returns `error`, `internal/modules/**` returns `Result`:
// the boundary is sharp and checkable. The taxonomy of `auth` therefore goes
// through enumerated sentinels in `domain/`, recognisable by `errors.Is`
// (ADR 017).
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// Name is the module name, as it appears in config/modules.yaml.
const Name = "auth"

// Available drivers.
const (
	driverMemory = "memory"
)

// Option keys read by this module, shared with the catalogue (ADR 014, #93).
const (
	// OptionSessionTTL bounds the lifetime of a session.
	OptionSessionTTL = "session_ttl"
)

// defaultSessionTTL is the session lifetime when the configuration says nothing.
//
// Twelve hours: long enough for a working day without reconnecting, short
// enough that a forgotten workstation stops opening the door the next day. This
// is not a security bound — revocation, on the other hand, is immediate
// (ADR 017 §1).
const defaultSessionTTL = 12 * time.Hour

// tokenBytes is the size of a token's randomness.
//
// 32 bytes, that is 256 bits. In base64 without padding that makes 43
// characters, which is what `domain.NewToken` demands.
const tokenBytes = 32

// Module exposes the primary ports.
//
// A surface — HTTP, CLI, event consumer — receives ONLY this structure. It
// therefore cannot reach the store, nor bypass a use case.
type Module struct {
	Register     ports.Register
	Authenticate ports.Authenticate
	Verify       ports.Verify
	Authorize    ports.Authorize
	Revoke       ports.Revoke
	DefineRole   ports.DefineRole
	AssignRoles  ports.AssignRoles
	Deactivate   ports.Deactivate
	Reactivate   ports.Reactivate
}

// Deps carries the effects the module does not build itself.
//
// Hashing is NOT provided by this module: it is costly, parameterised, and its
// tuning belongs to the application's security configuration
// (`internal/infrastructure/security`). A module that chose its own Argon2
// parameters would freeze them for everyone.
type Deps struct {
	HashSecret   ports.HashSecret
	VerifySecret ports.VerifySecret
	Now          ports.Now
}

// Module errors.
var (
	// ErrDisabled reports a call to a disabled module.
	ErrDisabled = errors.New("auth module disabled")

	// ErrMissingDependency refuses an incomplete wiring.
	ErrMissingDependency = errors.New("mandatory dependency missing")

	errUnknownDriver = errors.New("unknown auth driver")
)

// New builds the module according to the requested driver.
//
// An unknown driver REFUSES startup: a typo never resolves into "the closest
// driver". Deny by default.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return Disabled(), nil
	}
	if err := deps.validate(); err != nil {
		return Module{}, err
	}

	ttl, err := cfg.DurationOption(OptionSessionTTL, defaultSessionTTL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}

	driver := cfg.Driver
	if driver == "" {
		driver = driverMemory
	}
	switch driver {
	case driverMemory:
		return assemble(memory.New(), deps, ttl), nil
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, driver)
	}
}

// validate refuses an incomplete wiring BEFORE any construction.
//
// Without this refusal, a forgotten dependency would produce a nil pointer
// panic on the first real sign-in — so in production, and with a stack trace
// that does not say which one was missing.
func (d Deps) validate() error {
	missing := map[string]bool{
		"HashSecret":   d.HashSecret == nil,
		"VerifySecret": d.VerifySecret == nil,
		"Now":          d.Now == nil,
	}
	for name, absent := range missing {
		if absent {
			return fmt.Errorf("%w: %s", ErrMissingDependency, name)
		}
	}
	return nil
}

// store is what the module expects from a driver.
//
// Declared here and not in `ports/`: it is an internal composition convenience,
// not a public contract. The module's ports remain function types, and it is
// `assemble` that extracts the methods from it.
type store interface {
	SaveIdentity(context.Context, domain.Credential) error
	FindBySubject(context.Context, domain.Subject) (domain.Credential, error)
	FindIdentity(context.Context, domain.IdentityID) (domain.Identity, error)
	UpdateIdentity(context.Context, domain.Identity) error
	SaveSession(context.Context, domain.Session) error
	FindSession(context.Context, domain.Token) (domain.Session, error)
	DeleteSession(context.Context, domain.Token) error
	SaveRole(context.Context, domain.Role) error
	AssignRoles(context.Context, domain.IdentityID, []string) error
	Grants(context.Context, domain.IdentityID, domain.Permission) bool
}

// assemble wires a driver onto the use cases.
func assemble(s store, deps Deps, ttl time.Duration) Module {
	wired := application.Deps{
		SaveIdentity:   s.SaveIdentity,
		FindBySubject:  s.FindBySubject,
		FindIdentity:   s.FindIdentity,
		UpdateIdentity: s.UpdateIdentity,
		SaveSession:    s.SaveSession,
		FindSession:    s.FindSession,
		DeleteSession:  s.DeleteSession,
		Grants:         s.Grants,
		SaveRole:       s.SaveRole,
		BindRoles:      s.AssignRoles,
		HashSecret:     deps.HashSecret,
		VerifySecret:   deps.VerifySecret,
		Now:            deps.Now,
		NewToken:       randomToken,
		NewIdentityID:  randomIdentityID,
		SessionTTL:     ttl,
	}

	return Module{
		Register:     application.NewRegister(wired),
		Authenticate: application.NewAuthenticate(wired),
		Verify:       application.NewVerify(wired),
		Authorize:    application.NewAuthorize(wired),
		Revoke:       application.NewRevoke(wired),
		DefineRole:   application.NewDefineRole(wired),
		AssignRoles:  application.NewAssignRoles(wired),
		Deactivate:   application.NewDeactivate(wired),
		Reactivate:   application.NewReactivate(wired),
	}
}

// randomToken draws a token from a cryptographically secure source.
//
// `crypto/rand` and not `math/rand`: a predictable token is a bypassed
// authentication, and `math/rand` produces those — its seed fits in 63 bits,
// and its state can be reconstructed from a handful of outputs.
//
// The error is PROPAGATED, never swallowed: if the system's entropy is
// unavailable, the right answer is to refuse to issue a token, not to
// manufacture a weaker one.
func randomToken() (domain.Token, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return domain.Token{}, fmt.Errorf("entropy unavailable: %w", err)
	}
	token, err := domain.NewToken(base64.RawURLEncoding.EncodeToString(raw))
	if err != nil {
		return domain.Token{}, fmt.Errorf("token: %w", err)
	}
	return token, nil
}

// randomIdentityID produces an opaque identifier.
//
// The empty string IS returned when entropy fails — the body says so, three
// lines below. What is impossible is that this empty string becomes an
// identity: `domain.NewIdentity` refuses an empty identifier, so the failure
// surfaces as a creation error rather than as an anonymous identity.
//
// ⚠️ This comment used to state the fallback was "IMPOSSIBLE here", which the
// code contradicts on sight. The intent was about the consequence; the sentence
// denied the code.
func randomIdentityID() domain.IdentityID {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	return domain.IdentityID(base64.RawURLEncoding.EncodeToString(raw))
}

// Disabled returns a module that refuses on call.
//
// It always mounts: that is what lets a surface exist and answer a clear error,
// rather than failing the whole server startup for a module nobody enabled.
//
// ⚠️ `Authorize` refuses too. A disabled authentication module that would
// AUTHORISE everything would be the worst default value imaginable — and that
// is exactly what forgetting this case gets you.
func Disabled() Module {
	return Module{
		Register: func(context.Context, string, string) (domain.Identity, error) {
			return domain.Identity{}, ErrDisabled
		},
		Authenticate: func(context.Context, string, string) (domain.Session, error) {
			return domain.Session{}, ErrDisabled
		},
		Verify: func(context.Context, domain.Token) (domain.Identity, error) {
			return domain.Identity{}, ErrDisabled
		},
		Authorize: func(context.Context, domain.IdentityID, domain.Permission) error {
			return ErrDisabled
		},
		Revoke:      func(context.Context, domain.Token) error { return ErrDisabled },
		DefineRole:  func(context.Context, string, []string) error { return ErrDisabled },
		AssignRoles: func(context.Context, domain.IdentityID, []string) error { return ErrDisabled },
		Deactivate:  func(context.Context, domain.IdentityID) error { return ErrDisabled },
		Reactivate:  func(context.Context, domain.IdentityID) error { return ErrDisabled },
	}
}
