// Package hashing branche le port HashPassword sur l'infrastructure de sécurité.
//
// Le hachage n'est PAS du domaine : c'est un effet coûteux, paramétré, et dont
// les paramètres changent avec le matériel. Le domaine ne connaît donc qu'un
// `PasswordHash` opaque, et ignore jusqu'au nom de l'algorithme.
//
// Conséquence pratique : passer d'Argon2id à ce qui lui succédera ne touche pas
// une ligne de `domain/` ni de `application/`.
package hashing

import (
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/ports"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// New construit le port de hachage.
//
// Contrat d'erreur : CodeInternal UNIQUEMENT. Un échec de hachage est un défaut
// technique — entropie indisponible, paramètres absurdes — jamais une erreur
// imputable à l'utilisateur. Le traduire en erreur de saisie ferait accuser un
// mot de passe pourtant valide.
func New(hasher security.Hasher) ports.HashPassword {
	return func(password domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
		// Expose() et NON String() : String() rend « [redacted] », par
		// conception, pour qu'aucune journalisation accidentelle ne fasse fuir le
		// mot de passe. L'utiliser ici hacherait la MÊME chaîne pour tous les
		// comptes — tous partageraient un condensé, et n'importe quel mot de
		// passe ouvrirait n'importe quel compte.
		//
		// C'est le seul endroit du dépôt autorisé à appeler Expose(), et le nom
		// est explicite pour que cet appel se voie en relecture.
		encoded, err := hasher.Hash(password.Expose())
		if err != nil {
			// Le message rendu ne nomme ni l'algorithme, ni la cause : les deux
			// renseignent un attaquant sur ce qui tourne derrière.
			return result.Err[domain.PasswordHash, domain.Error](
				domain.NewError(
					domain.CodeInternal,
					"le mot de passe n'a pas pu être traité",
				).WithCause(err),
			)
		}
		return result.Ok[domain.PasswordHash, domain.Error](domain.NewPasswordHash(encoded))
	}
}
