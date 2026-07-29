package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// BootstrapRole est le rôle accordé au compte d'amorçage.
//
// # Pourquoi l'amorçage accorde des droits, et pas seulement un compte
//
// Un compte sans rôle n'accorde RIEN — deny par défaut. L'amorçage produirait
// donc un administrateur qui reçoit 403 sur chaque route d'administration : le
// délai avant premier succès resterait infini, déplacé d'un cran.
//
// Les permissions accordées sont celles de la surface d'administration, et rien
// de plus. En particulier, ce rôle ne présume d'aucun droit MÉTIER : une
// application qui monte ce socle compose les siens.
const BootstrapRole = "admin"

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
//  4. **Il ne fait jamais échouer un démarrage.** Un module désactivé n'est pas
//     une panne, c'est un état configuré : il n'y a rien à amorcer, et le dire
//     par une erreur fatale empêcherait de démarrer tout environnement local où
//     l'authentification est éteinte.
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

	identity, err := mod.Register(ctx, BootstrapSubject, secret)
	if err != nil {
		if errors.Is(err, domain.ErrSubjectTaken) || errors.Is(err, ErrDisabled) {
			// Deux « rien à faire » distincts, et le même compte rendu vide :
			//
			//   ErrSubjectTaken — déjà amorcé. On ne recrée pas, et on ne rend
			//   pas le secret d'un compte dont on ne connaît plus le mot de
			//   passe.
			//
			//   ErrDisabled — le module est ÉTEINT. C'est un état configuré,
			//   pas une panne : il n'y a simplement rien à amorcer.
			//
			// ⚠️ Le second cas remontait comme une erreur fatale, et FAISAIT
			// ÉCHOUER LE DÉMARRAGE dès qu'`auth` était désactivé. Il l'était en
			// `test` — `IsLocal()` couvre `development` ET `test`, alors que
			// l'activation ne vient que de la couche `development`. Trouvé par
			// la CI end-to-end, jamais en local : la mesure locale avait porté
			// sur `development` et `production`, donc jamais sur la seule
			// combinaison qui casse, environnement local ET module éteint.
			return BootstrapReport{}, nil
		}
		return BootstrapReport{}, fmt.Errorf("amorçage du compte %q: %w", BootstrapSubject, err)
	}

	if err := grantAdmin(ctx, mod, identity.ID); err != nil {
		return BootstrapReport{}, err
	}
	return BootstrapReport{Created: true, Subject: BootstrapSubject, Secret: secret}, nil
}

// grantAdmin définit le rôle d'amorçage et l'accorde.
//
// # L'ordre compte, et l'échec doit être bruyant
//
// Définir avant d'affecter : l'inverse fonctionnerait — un rôle affecté avant
// d'exister n'accorde rien puis accorde — mais laisserait une fenêtre pendant
// laquelle le compte annoncé comme administrateur ne peut rien faire.
//
// Un échec ici REMONTE plutôt que d'être avalé. Rendre un compte rendu « créé »
// sur un compte sans droits produirait le pire message possible : l'exploitant
// lit un secret, se connecte, et reçoit 403 partout sans savoir pourquoi.
func grantAdmin(ctx context.Context, mod Module, id domain.IdentityID) error {
	permissions := []string{
		contract.PermissionIdentityCreate,
		contract.PermissionIdentityRoles,
		contract.PermissionIdentityClose,
		contract.PermissionRoleWrite,
	}
	if err := mod.DefineRole(ctx, BootstrapRole, permissions); err != nil {
		return fmt.Errorf("définition du rôle %q: %w", BootstrapRole, err)
	}
	if err := mod.AssignRoles(ctx, id, []string{BootstrapRole}); err != nil {
		return fmt.Errorf("affectation du rôle %q: %w", BootstrapRole, err)
	}
	return nil
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
