package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// BootstrapSubject désigne le compte d'amorçage.
//
// Constante et non configurable : un sujet paramétrable inviterait à écrire le
// compte administrateur de production dans un fichier versionné, et c'est
// exactement ce qu'on veut rendre impossible. Ici, le nom dit ce qu'il est —
// local, jetable, sans valeur ailleurs.
const BootstrapSubject = "admin@local"

// bootstrapSecretBytes est la taille de l'aléa du secret engendré.
//
// 24 octets, soit 32 caractères en base64 — bien au-delà des douze exigés, et
// assez court pour être recopié depuis un journal sans se tromper.
const bootstrapSecretBytes = 24

// BootstrapReport dit ce que l'amorçage a fait, sans le journaliser lui-même.
//
// Le module ne journalise pas (`application/` non plus) : il rend compte. C'est
// l'appelant qui décide du niveau, du format et de la destination — et c'est ce
// qui permet de tester l'amorçage sans analyser des journaux.
type BootstrapReport struct {
	// Created dit si un compte a été créé PAR CET APPEL.
	Created bool

	// Subject est le compte d'amorçage, vide si rien n'a été fait.
	Subject string

	// Secret est le secret ENGENDRÉ, à afficher une seule fois.
	//
	// ⚠️ Il ne sera plus jamais lisible : seul son condensé est retenu. Le champ
	// est vide dès que `Created` est faux.
	Secret string
}

// Bootstrap crée un compte d'amorçage — EN DÉVELOPPEMENT UNIQUEMENT.
//
// # Le problème que ceci résout
//
// La surface d'authentification ne publie aucune opération d'administration :
// les exposer sans les protéger ouvrirait la création de comptes à quiconque, et
// les protéger exige un premier administrateur. Un serveur neuf rendait donc
// **401 à tout le monde**, sans exception — le délai avant premier succès du
// module était infini (#99).
//
// # Ce qui rend ce raccourci acceptable, et lui seul
//
//  1. **Il ne s'applique qu'en local.** Hors `development` et `test`, la fonction
//     ne crée RIEN et le dit dans son compte rendu. Ce n'est pas une erreur —
//     faire échouer le démarrage d'une production parce qu'elle refuse un compte
//     de démonstration serait absurde — c'est un refus d'agir.
//  2. **Le secret est ENGENDRÉ, jamais écrit.** Aucun mot de passe par défaut
//     n'existe dans un artefact versionné. C'est la faute qui compte vraiment
//     ici : un socle livré avec `admin/admin` est un socle qui déploie
//     `admin/admin`, et personne ne le change avant l'incident.
//  3. **Il est idempotent.** Un sujet déjà pris n'est pas une erreur, et rien
//     n'est recréé : redémarrer ne réinitialise pas un compte existant.
//
// # Pourquoi le secret est rendu plutôt que journalisé ici
//
// Parce qu'un module noyau ne journalise pas. L'appelant le reçoit et décide —
// et cette frontière est ce qui garantit qu'un secret ne part pas dans un
// collecteur d'observabilité parce qu'un module a cru bien faire.
func Bootstrap(ctx context.Context, mod Module, env config.Environment) (BootstrapReport, error) {
	if !env.IsLocal() {
		// Refus d'agir, pas erreur. Le compte rendu vide EST la réponse.
		return BootstrapReport{}, nil
	}

	secret, err := randomSecret()
	if err != nil {
		return BootstrapReport{}, err
	}

	if _, err := mod.Register(ctx, BootstrapSubject, secret); err != nil {
		if errors.Is(err, domain.ErrSubjectTaken) {
			// Déjà amorcé : on ne recrée pas, et on ne rend pas le secret d'un
			// compte dont on ne connaît pas le mot de passe.
			return BootstrapReport{}, nil
		}
		return BootstrapReport{}, fmt.Errorf("amorçage du compte %q: %w", BootstrapSubject, err)
	}

	return BootstrapReport{Created: true, Subject: BootstrapSubject, Secret: secret}, nil
}

// randomSecret tire un secret d'une source cryptographiquement sûre.
//
// `crypto/rand` et non `math/rand`, pour la même raison que les jetons : un
// secret prévisible est une authentification contournée. Le fait qu'il soit
// « seulement » de développement ne change rien — un poste de développement est
// souvent joignable depuis le réseau local.
func randomSecret() (string, error) {
	raw := make([]byte, bootstrapSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("entropie indisponible: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
