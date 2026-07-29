// Package userregistration is the composition root of the business module.
//
// It is the ONLY file in the module that knows the drivers (ADR 012). The use
// case, for its part, only sees function types: changing driver does not touch a
// single line of `application/` or `domain/`.
//
// # This module is the REFERENCE SLICE of the starter
//
// It is not "the application". It exists to show the complete shape of a
// business module — pure domain, ports as function types, composed pipeline,
// interchangeable drivers, one adapter per surface — because that shape is the
// one that will be copied to write `billing` or `crm`.
//
// `hexa new` must be able to delete it with a single `rm -rf` without breaking
// anything else: no starter code names it.
package userregistration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Name is the name of the module.
const Name = "user_registration"

// Available persistence drivers.
//
// `memory` is the default, and that is a decision: it requires no
// infrastructure, so `go run` starts. See the NON-GUARANTEES of the
// drivers/memory package before considering it anywhere other than development.
const (
	DriverMemory = "memory"
)

// Module exposes the primary ports of the module.
//
// A surface — HTTP, CLI, event consumer — receives ONLY this structure. It
// therefore cannot reach a driver, nor open a transaction, nor bypass the use
// case.
type Module struct {
	Register   ports.RegisterUser
	CheckEmail ports.CheckEmailAvailability
}

// Deps carries the effects the module does not build itself.
//
// All of them are function types: in a test, each one is a three-line closure,
// and no mocking library is needed — therefore none is allowed
// (rules/dependances.md).
type Deps struct {
	// HashPassword comes from the security infrastructure: hashing is a
	// costly, parameterised effect, never domain.
	HashPassword ports.HashPassword
	// PublishEvent writes into the outbox, WITHIN the current transaction
	// (ADR 006).
	PublishEvent ports.PublishEvent
	// GenerateID and Now are ports so that tests are deterministic.
	GenerateID ports.GenerateID
	Now        ports.Now
}

// Errors of the module.
var (
	// ErrDisabled reports a call to a disabled module.
	//
	// A disabled module fails explicitly rather than falling back on inert
	// behaviour: a silently ignored registration is the worst possible defect,
	// because it never reports itself.
	ErrDisabled = errors.New("module user_registration disabled")

	// ErrMissingDependency refuses an incomplete assembly.
	ErrMissingDependency = errors.New("mandatory dependency missing")

	errUnknownDriver = errors.New("unknown user_registration driver")
)

// New builds the module according to the requested driver.
//
// An unknown driver REFUSES to start: a typo is never resolved into "the closest
// driver". Deny by default.
func New(driver string, deps Deps) (Module, error) {
	if err := deps.validate(); err != nil {
		return Module{}, err
	}
	if driver == "" {
		driver = DriverMemory
	}

	switch driver {
	case DriverMemory:
		return assemble(memory.New(), deps), nil
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, driver)
	}
}

// validate refuses an incomplete assembly BEFORE any construction.
//
// Without this refusal, a forgotten dependency would produce a nil pointer panic
// on the first real request — therefore in production, and with a stack trace
// that does not say which one was missing.
func (d Deps) validate() error {
	missing := map[string]bool{
		"HashPassword": d.HashPassword == nil,
		"PublishEvent": d.PublishEvent == nil,
		"GenerateID":   d.GenerateID == nil,
		"Now":          d.Now == nil,
	}
	for name, absent := range missing {
		if absent {
			return fmt.Errorf("%w: %s", ErrMissingDependency, name)
		}
	}
	return nil
}

// store is what the module expects from a persistence driver.
//
// Declared here and not in `ports/`: it is an internal composition convenience,
// not a public contract. The module's ports remain function types, and it is
// `assemble` that extracts the methods from it.
type store interface {
	Save(context.Context, domain.User) result.Result[domain.User, domain.Error]
	IsTaken(context.Context, domain.Email) result.Result[bool, domain.Error]
}

// assemble wires a driver onto the use cases.
func assemble(s store, deps Deps) Module {
	register := application.NewRegisterUser(application.Deps{
		EmailIsTaken: s.IsTaken,
		HashPassword: deps.HashPassword,
		SaveUser:     s.Save,
		PublishEvent: deps.PublishEvent,
		GenerateID:   deps.GenerateID,
		Now:          deps.Now,
	})

	return Module{
		Register:   register,
		CheckEmail: application.NewCheckEmailAvailability(s.IsTaken),
	}
}

// Disabled returns a module that refuses when called.
//
// It always mounts: that is what allows the HTTP surface to exist and answer a
// clear error, rather than failing the whole server start-up for a module nobody
// enabled.
func Disabled() Module {
	failRegister := func(context.Context, domain.RegistrationCommand) result.Result[domain.User, domain.Error] {
		return result.Err[domain.User, domain.Error](disabledError())
	}
	failCheck := func(context.Context, string) result.Result[bool, domain.Error] {
		return result.Err[bool, domain.Error](disabledError())
	}
	return Module{Register: failRegister, CheckEmail: failCheck}
}

func disabledError() domain.Error {
	return domain.NewError(
		domain.CodeUnavailable,
		"l'inscription est indisponible",
	).WithCause(ErrDisabled)
}

// SystemClock is the real clock, to be passed in production.
//
// Named rather than written `time.Now` at the call site: the composition root is
// the ONLY place in the repository allowed to read the clock, and giving it a
// name makes the derogation visible in review.
func SystemClock() ports.Now { return time.Now }
