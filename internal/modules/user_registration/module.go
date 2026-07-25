// Package userregistration est la composition root du module métier.
//
// C'est le SEUL fichier du module qui connaît les pilotes (ADR 012). Le cas
// d'usage, lui, ne voit que des types fonction : changer de pilote ne touche pas
// une ligne de `application/` ni de `domain/`.
//
// # Ce module est la TRANCHE DE RÉFÉRENCE du socle
//
// Il n'est pas « l'application ». Il existe pour montrer la forme complète d'un
// module métier — domaine pur, ports en types fonction, pipeline composé,
// pilotes interchangeables, adaptateurs par surface — parce que c'est cette
// forme qui sera copiée pour écrire `billing` ou `crm`.
//
// `hexa new` doit pouvoir le supprimer d'un seul `rm -rf` sans rien casser
// d'autre : aucun code du socle ne le nomme.
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

// Name est le nom du module.
const Name = "user_registration"

// Pilotes de persistance disponibles.
//
// `memory` est le défaut, et c'est une décision : il n'exige aucune
// infrastructure, donc `go run` démarre. Voir les NON-garanties du paquet
// drivers/memory avant de l'envisager ailleurs qu'en développement.
const (
	DriverMemory = "memory"
)

// Module expose les ports primaires du module.
//
// Une surface — HTTP, CLI, consommateur d'événements — ne reçoit QUE cette
// structure. Elle ne peut donc pas atteindre un pilote, ni ouvrir une
// transaction, ni contourner le cas d'usage.
type Module struct {
	Register   ports.RegisterUser
	CheckEmail ports.CheckEmailAvailability
}

// Deps porte les effets que le module ne fabrique pas lui-même.
//
// Tous sont des types fonction : en test, chacun est une closure de trois
// lignes, et aucune bibliothèque de simulacre n'est nécessaire — donc aucune
// n'est autorisée (rules/dependances.md).
type Deps struct {
	// HashPassword vient de l'infrastructure de sécurité : le hachage est un
	// effet coûteux et paramétré, jamais du domaine.
	HashPassword ports.HashPassword
	// PublishEvent écrit dans l'outbox, DANS la transaction courante (ADR 006).
	PublishEvent ports.PublishEvent
	// GenerateID et Now sont des ports pour que les tests soient déterministes.
	GenerateID ports.GenerateID
	Now        ports.Now
}

// Erreurs du module.
var (
	// ErrDisabled signale un appel à un module désactivé.
	//
	// Un module désactivé échoue explicitement plutôt que de se rabattre sur un
	// comportement inerte : une inscription silencieusement ignorée est le pire
	// défaut possible, parce qu'elle ne se signale jamais.
	ErrDisabled = errors.New("module user_registration désactivé")

	// ErrMissingDependency refuse un montage incomplet.
	ErrMissingDependency = errors.New("dépendance obligatoire absente")

	errUnknownDriver = errors.New("pilote user_registration inconnu")
)

// New construit le module selon le pilote demandé.
//
// Un pilote inconnu REFUSE le démarrage : une faute de frappe ne se résout
// jamais en « le pilote le plus proche ». Deny par défaut.
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

// validate refuse un montage incomplet AVANT toute construction.
//
// Sans ce refus, une dépendance oubliée produirait un panic de pointeur nil à la
// première requête réelle — donc en production, et avec une trace qui ne dit pas
// laquelle manquait.
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

// store est ce que le module attend d'un pilote de persistance.
//
// Déclaré ici et non dans `ports/` : c'est une commodité de composition interne,
// pas un contrat public. Les ports du module restent des types fonction, et
// c'est `assemble` qui en extrait les méthodes.
type store interface {
	Save(context.Context, domain.User) result.Result[domain.User, domain.Error]
	IsTaken(context.Context, domain.Email) result.Result[bool, domain.Error]
}

// assemble branche un pilote sur les cas d'usage.
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

// Disabled rend un module qui refuse à l'appel.
//
// Il se monte toujours : c'est ce qui permet à la surface HTTP d'exister et de
// répondre une erreur claire, plutôt que de faire échouer le démarrage entier du
// serveur pour un module que personne n'a activé.
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

// SystemClock est l'horloge réelle, à passer en production.
//
// Nommée plutôt qu'écrite `time.Now` au site d'appel : le composition root est
// le SEUL endroit du dépôt autorisé à lire l'horloge, et lui donner un nom rend
// la dérogation visible en relecture.
func SystemClock() ports.Now { return time.Now }
