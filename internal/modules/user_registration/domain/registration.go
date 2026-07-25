package domain

import "github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"

// RegistrationCommand est une intention d'inscription telle qu'elle arrive d'une
// surface : des types primitifs, non encore validés.
//
// C'est le SEUL endroit du domaine où l'on trouve des `string` nues, et c'est
// justement le rôle de ce type : marquer la frontière.
type RegistrationCommand struct {
	Email    string
	Password string
}

// ValidRegistration est une intention d'inscription validée.
//
// Une fois ce type obtenu, plus aucune validation n'est nécessaire en aval : le
// franchissement de la frontière a eu lieu, une fois, ici.
type ValidRegistration struct {
	Email    Email
	Password RawPassword
}

// ParseRegistration valide une commande d'inscription.
//
// Fonction pure : aucune I/O, aucune horloge. L'unicité de l'adresse n'est PAS
// vérifiée ici — c'est un effet, donc un port, appelé par le cas d'usage.
func ParseRegistration(cmd RegistrationCommand) result.Result[ValidRegistration, Error] {
	// L'ordre compte : on signale l'adresse avant le mot de passe, parce que
	// c'est le champ que l'utilisateur remplit en premier.
	return result.FlatMap(
		NewEmail(cmd.Email),
		func(email Email) result.Result[ValidRegistration, Error] {
			return result.Map(
				NewRawPassword(cmd.Password),
				func(password RawPassword) ValidRegistration {
					return ValidRegistration{Email: email, Password: password}
				},
			)
		},
	)
}
