package application

import (
	"context"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/ports"
)

// NewAuthenticate compose l'échange d'un secret contre une session.
//
// # L'ordre des étapes est la décision, et il se lit
//
// Valider la forme, retrouver la créance, comparer le secret, émettre la session.
// Une inversion ferait émettre avant de comparer — c'est-à-dire authentifier
// n'importe qui.
//
// # Un seul refus pour trois causes, délibérément
//
// Sujet mal formé, sujet inconnu, secret faux : tout rend
// `domain.ErrInvalidCredentials`. Distinguer « ce compte n'existe pas » de « le
// mot de passe est faux » transforme le formulaire de connexion en **oracle
// d'existence de comptes** — on interroge mille adresses, on note lesquelles
// répondent différemment, et on sait où concentrer l'effort.
//
// ⚠️ Cette fonction reste néanmoins un oracle par le TEMPS : un sujet inconnu
// répond sans hachage, donc plus vite. Le remède est de hacher quand même, et il
// n'est pas appliqué ici — voir la NON-garantie écrite à la fin de ce fichier.
func NewAuthenticate(deps Deps) ports.Authenticate {
	return func(ctx context.Context, rawSubject, secret string) (domain.Session, error) {
		subject, err := domain.NewSubject(rawSubject)
		if err != nil {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		credential, err := deps.FindBySubject(ctx, subject)
		if err != nil {
			return domain.Session{}, fmt.Errorf("authentification: %w", err)
		}
		if !credential.Identity().Active {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		matches, err := deps.VerifySecret(secret, credential.SecretHash())
		if err != nil {
			return domain.Session{}, fmt.Errorf("comparaison du secret: %w", err)
		}
		if !matches {
			return domain.Session{}, domain.ErrInvalidCredentials
		}

		return deps.issue(ctx, credential.Identity().ID)
	}
}

// issue produit et persiste une session.
//
// Isolée pour que `NewAuthenticate` reste sous le seuil de lignes d'`arch-go`, et
// parce qu'elle nomme le seul endroit du module où un jeton naît.
func (d Deps) issue(ctx context.Context, id domain.IdentityID) (domain.Session, error) {
	token, err := d.NewToken()
	if err != nil {
		return domain.Session{}, fmt.Errorf("production du jeton: %w", err)
	}

	session, err := domain.NewSession(token, id, d.Now(), d.SessionTTL)
	if err != nil {
		return domain.Session{}, fmt.Errorf("ouverture de session: %w", err)
	}
	if err := d.SaveSession(ctx, session); err != nil {
		return domain.Session{}, fmt.Errorf("enregistrement de la session: %w", err)
	}
	return session, nil
}

// NON-GARANTIE — l'oracle temporel n'est pas fermé.
//
// Un sujet inconnu rend `ErrInvalidCredentials` SANS avoir haché quoi que ce
// soit ; un sujet connu paie un Argon2id complet, soit plusieurs dizaines de
// millisecondes. La différence est mesurable sur un réseau local, et elle révèle
// quels comptes existent — exactement ce que le message unique cherche à cacher.
//
// Le remède est connu : hacher contre un condensé factice quand le sujet est
// inconnu, pour que les deux chemins coûtent pareil. Il n'est PAS appliqué ici
// parce qu'il exige un condensé de référence produit avec les mêmes paramètres
// que ceux configurés, et que le fabriquer au démarrage est une décision
// d'infrastructure, pas de cas d'usage.
//
// Écrit plutôt que tu : une faiblesse nommée se corrige, une faiblesse ignorée
// se découvre en audit.
