// Package auth est la composition root du module d'authentification.
//
// C'est le SEUL fichier du module qui connaît les pilotes (ADR 012). Les cas
// d'usage ne voient que des types fonction : changer de pilote ne touche pas une
// ligne d'`application/` ni de `domain/`.
//
// # Ce module rend une `error`, pas un `Result`
//
// `internal/core/**` retourne `error`, `internal/modules/**` retourne `Result` :
// la frontière est nette et vérifiable. La taxonomie d'`auth` passe donc par des
// sentinelles énumérées de `domain/`, reconnaissables par `errors.Is` (ADR 017).
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

// Name est le nom du module, tel qu'il apparaît dans config/modules.yaml.
const Name = "auth"

// Pilotes disponibles.
const (
	driverMemory = "memory"
)

// Clés d'options lues par ce module, partagées avec le catalogue (ADR 014, #93).
const (
	// OptionSessionTTL borne la durée d'une session.
	OptionSessionTTL = "session_ttl"
)

// defaultSessionTTL est la durée d'une session quand la configuration n'en dit rien.
//
// Douze heures : assez pour une journée de travail sans reconnexion, assez court
// pour qu'un poste oublié cesse d'ouvrir la porte le lendemain. Ce n'est pas une
// borne de sécurité — la révocation, elle, est immédiate (ADR 017 §1).
const defaultSessionTTL = 12 * time.Hour

// tokenBytes est la taille de l'aléa d'un jeton.
//
// 32 octets, soit 256 bits. En base64 sans remplissage cela fait 43 caractères,
// ce que `domain.NewToken` exige.
const tokenBytes = 32

// Module expose les ports primaires.
//
// Une surface — HTTP, CLI, consommateur d'événements — ne reçoit QUE cette
// structure. Elle ne peut donc pas atteindre le magasin, ni contourner un cas
// d'usage.
type Module struct {
	Register     ports.Register
	Authenticate ports.Authenticate
	Verify       ports.Verify
	Authorize    ports.Authorize
	Revoke       ports.Revoke
	DefineRole   ports.DefineRole
	AssignRoles  ports.AssignRoles
}

// Deps porte les effets que le module ne fabrique pas lui-même.
//
// Le hachage n'est PAS fourni par ce module : il est coûteux, paramétré, et son
// réglage appartient à la configuration de sécurité de l'application
// (`internal/infrastructure/security`). Un module qui choisirait ses propres
// paramètres Argon2 les figerait pour tout le monde.
type Deps struct {
	HashSecret   ports.HashSecret
	VerifySecret ports.VerifySecret
	Now          ports.Now
}

// Erreurs du module.
var (
	// ErrDisabled signale un appel à un module désactivé.
	ErrDisabled = errors.New("module auth désactivé")

	// ErrMissingDependency refuse un montage incomplet.
	ErrMissingDependency = errors.New("dépendance obligatoire absente")

	errUnknownDriver = errors.New("pilote auth inconnu")
)

// New construit le module selon le pilote demandé.
//
// Un pilote inconnu REFUSE le démarrage : une faute de frappe ne se résout jamais
// en « le pilote le plus proche ». Deny par défaut.
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

// validate refuse un montage incomplet AVANT toute construction.
//
// Sans ce refus, une dépendance oubliée produirait un panic de pointeur nil à la
// première connexion réelle — donc en production, et avec une trace qui ne dit
// pas laquelle manquait.
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

// store est ce que le module attend d'un pilote.
//
// Déclaré ici et non dans `ports/` : c'est une commodité de composition interne,
// pas un contrat public. Les ports du module restent des types fonction, et c'est
// `assemble` qui en extrait les méthodes.
type store interface {
	SaveIdentity(context.Context, domain.Credential) error
	FindBySubject(context.Context, domain.Subject) (domain.Credential, error)
	FindIdentity(context.Context, domain.IdentityID) (domain.Identity, error)
	SaveSession(context.Context, domain.Session) error
	FindSession(context.Context, domain.Token) (domain.Session, error)
	DeleteSession(context.Context, domain.Token) error
	SaveRole(context.Context, domain.Role) error
	AssignRoles(context.Context, domain.IdentityID, []string) error
	Grants(context.Context, domain.IdentityID, domain.Permission) bool
}

// assemble branche un pilote sur les cas d'usage.
func assemble(s store, deps Deps, ttl time.Duration) Module {
	wired := application.Deps{
		SaveIdentity:  s.SaveIdentity,
		FindBySubject: s.FindBySubject,
		FindIdentity:  s.FindIdentity,
		SaveSession:   s.SaveSession,
		FindSession:   s.FindSession,
		DeleteSession: s.DeleteSession,
		Grants:        s.Grants,
		SaveRole:      s.SaveRole,
		BindRoles:     s.AssignRoles,
		HashSecret:    deps.HashSecret,
		VerifySecret:  deps.VerifySecret,
		Now:           deps.Now,
		NewToken:      randomToken,
		NewIdentityID: randomIdentityID,
		SessionTTL:    ttl,
	}

	return Module{
		Register:     application.NewRegister(wired),
		Authenticate: application.NewAuthenticate(wired),
		Verify:       application.NewVerify(wired),
		Authorize:    application.NewAuthorize(wired),
		Revoke:       application.NewRevoke(wired),
		DefineRole:   application.NewDefineRole(wired),
		AssignRoles:  application.NewAssignRoles(wired),
	}
}

// randomToken tire un jeton d'une source cryptographiquement sûre.
//
// `crypto/rand` et non `math/rand` : un jeton prévisible est une
// authentification contournée, et `math/rand` en produit — sa graine tient sur
// 63 bits, et son état se reconstitue à partir de quelques sorties.
//
// L'erreur est PROPAGÉE, jamais avalée : si l'entropie du système est
// indisponible, la bonne réponse est de refuser d'émettre un jeton, pas d'en
// fabriquer un moins bon.
func randomToken() (domain.Token, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return domain.Token{}, fmt.Errorf("entropie indisponible: %w", err)
	}
	token, err := domain.NewToken(base64.RawURLEncoding.EncodeToString(raw))
	if err != nil {
		return domain.Token{}, fmt.Errorf("jeton: %w", err)
	}
	return token, nil
}

// randomIdentityID produit un identifiant opaque.
//
// Le repli sur une chaîne vide en cas d'échec d'entropie est IMPOSSIBLE ici :
// `domain.NewIdentity` refuse un identifiant vide, donc l'échec remonte comme
// une erreur de création plutôt que comme une identité anonyme.
func randomIdentityID() domain.IdentityID {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	return domain.IdentityID(base64.RawURLEncoding.EncodeToString(raw))
}

// Disabled rend un module qui refuse à l'appel.
//
// Il se monte toujours : c'est ce qui permet à une surface d'exister et de
// répondre une erreur claire, plutôt que de faire échouer le démarrage entier du
// serveur pour un module que personne n'a activé.
//
// ⚠️ `Authorize` refuse lui aussi. Un module d'authentification désactivé qui
// AUTORISERAIT tout serait la pire valeur par défaut imaginable — et c'est
// exactement ce qu'on obtient en oubliant ce cas.
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
	}
}
