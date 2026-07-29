// Package application orchestrates the use cases and carries the decorators.
//
// It knows neither transport, nor persistence, nor a logger: it receives ports
// and returns Results. Enforced by arch-go.yml.
package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Deps gathers the effects the use case needs.
//
// Every field is a function type: a test double is a three-line closure, and no
// mocking library is needed — therefore none is allowed
// (rules/dependances.md).
type Deps struct {
	EmailIsTaken ports.EmailIsTaken
	HashPassword ports.HashPassword
	SaveUser     ports.SaveUser
	PublishEvent ports.PublishEvent
	GenerateID   ports.GenerateID
	Now          ports.Now
}

// state carries the intermediate state of the pipeline.
//
// Every step has the SAME signature `func(state) Result[state, Error]`, which
// allows composing them with result.Chain. Without that, six FlatMap calls would
// have to be nested — Go having no do-notation (documentation/adr/002).
type state struct {
	command      domain.RegistrationCommand
	registration domain.ValidRegistration
	hash         domain.PasswordHash
	user         domain.User
}

type step = func(state) result.Result[state, domain.Error]

// NewRegisterUser builds the registration use case.
//
// The body reads like the list of business steps, in order. Every step is
// testable in isolation, and the short circuit on the first failure is carried
// by result.Chain, not by repeated `if err != nil`.
func NewRegisterUser(deps Deps) ports.RegisterUser {
	return func(ctx context.Context, cmd domain.RegistrationCommand) result.Result[domain.User, domain.Error] {
		start := result.Ok[state, domain.Error](state{command: cmd})

		final := result.Chain(start,
			validate,
			deps.ensureEmailAvailable(ctx),
			deps.hashPassword,
			deps.buildUser,
			deps.persist(ctx),
			deps.publish(ctx),
		)

		return result.Map(final, func(s state) domain.User { return s.user })
	}
}

// validate is pure: it touches nothing on the outside.
func validate(s state) result.Result[state, domain.Error] {
	return result.Map(
		domain.ParseRegistration(s.command),
		func(registration domain.ValidRegistration) state {
			s.registration = registration
			return s
		},
	)
}

// ensureEmailAvailable closes over the context by partial application: the
// returned step has the homogeneous signature result.Chain requires.
func (d Deps) ensureEmailAvailable(ctx context.Context) step {
	return func(s state) result.Result[state, domain.Error] {
		return result.FlatMap(
			d.EmailIsTaken(ctx, s.registration.Email),
			func(taken bool) result.Result[state, domain.Error] {
				if taken {
					return result.Err[state, domain.Error](domain.NewError(
						domain.CodeEmailAlreadyExists,
						"cette adresse de courriel est déjà enregistrée",
					).WithField("email"))
				}
				return result.Ok[state, domain.Error](s)
			},
		)
	}
}

// hashPassword delegates to the port: hashing is an effect, never domain.
func (d Deps) hashPassword(s state) result.Result[state, domain.Error] {
	return result.Map(
		d.HashPassword(s.registration.Password),
		func(hash domain.PasswordHash) state {
			s.hash = hash
			return s
		},
	)
}

// buildUser is pure: the clock and the identifier come from ports.
func (d Deps) buildUser(s state) result.Result[state, domain.Error] {
	s.user = domain.NewUser(d.GenerateID(), s.registration.Email, s.hash, d.Now())
	return result.Ok[state, domain.Error](s)
}

func (d Deps) persist(ctx context.Context) step {
	return func(s state) result.Result[state, domain.Error] {
		return result.Map(d.SaveUser(ctx, s.user), func(saved domain.User) state {
			s.user = saved
			return s
		})
	}
}

// publish writes the event into the outbox, WITHIN the same transaction as the
// business write. A failure here cancels the registration: that is intended — a
// user created without their welcome event is a silently inconsistent state, and
// silence is the worst kind of defect.
func (d Deps) publish(ctx context.Context) step {
	return func(s state) result.Result[state, domain.Error] {
		event := domain.NewUserRegistered(s.user)
		return result.Map(
			d.PublishEvent(ctx, domain.EventUserRegistered, s.user.ID.String(), event),
			func(domain.Ack) state { return s },
		)
	}
}
