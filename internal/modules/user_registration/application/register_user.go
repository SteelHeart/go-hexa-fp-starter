// Package application orchestre les cas d'usage et porte les décorateurs.
//
// Il ne connaît ni transport, ni persistance, ni logger : il reçoit des ports et
// retourne des Result. Vérifié par .arch-go.yml.
package application

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Deps rassemble les effets dont le cas d'usage a besoin.
//
// Chaque champ est un type fonction : un double de test est une closure de trois
// lignes, et aucune bibliothèque de mock n'est nécessaire — donc aucune n'est
// autorisée (rules/dependances.md).
type Deps struct {
	EmailIsTaken ports.EmailIsTaken
	HashPassword ports.HashPassword
	SaveUser     ports.SaveUser
	PublishEvent ports.PublishEvent
	GenerateID   ports.GenerateID
	Now          ports.Now
}

// state porte l'état intermédiaire du pipeline.
//
// Toutes les étapes ont la MÊME signature `func(state) Result[state, Error]`, ce
// qui permet de les composer avec result.Chain. Sans cela, il faudrait imbriquer
// six FlatMap — Go n'ayant pas de do-notation (documentation/adr/002).
type state struct {
	command      domain.RegistrationCommand
	registration domain.ValidRegistration
	hash         domain.PasswordHash
	user         domain.User
}

type step = func(state) result.Result[state, domain.Error]

// NewRegisterUser construit le cas d'usage d'inscription.
//
// Le corps se lit comme la liste des étapes métier, dans l'ordre. Chaque étape
// est testable isolément, et le court-circuit au premier échec est porté par
// result.Chain, pas par des `if err != nil` répétés.
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

// validate est pure : elle ne touche à rien d'extérieur.
func validate(s state) result.Result[state, domain.Error] {
	return result.Map(
		domain.ParseRegistration(s.command),
		func(registration domain.ValidRegistration) state {
			s.registration = registration
			return s
		},
	)
}

// ensureEmailAvailable ferme le contexte par application partielle : l'étape
// retournée a la signature homogène exigée par result.Chain.
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

// hashPassword délègue au port : le hachage est un effet, jamais du domaine.
func (d Deps) hashPassword(s state) result.Result[state, domain.Error] {
	return result.Map(
		d.HashPassword(s.registration.Password),
		func(hash domain.PasswordHash) state {
			s.hash = hash
			return s
		},
	)
}

// buildUser est pure : l'horloge et l'identifiant viennent de ports.
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

// publish écrit l'événement dans l'outbox, DANS la même transaction que
// l'écriture métier. Un échec ici annule l'inscription : c'est voulu — un
// utilisateur créé sans son événement de bienvenue est un état incohérent
// silencieux, et le silence est le pire des défauts.
func (d Deps) publish(ctx context.Context) step {
	return func(s state) result.Result[state, domain.Error] {
		event := domain.NewUserRegistered(s.user)
		return result.Map(
			d.PublishEvent(ctx, domain.EventUserRegistered, s.user.ID.String(), event),
			func(struct{}) state { return s },
		)
	}
}
