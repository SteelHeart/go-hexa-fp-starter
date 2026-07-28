// Package notification est la composition root du module de notification.
//
// C'est le SEUL fichier du module qui connaît les pilotes (ADR 012). Les cas
// d'usage ne voient que des types fonction : changer de pilote ne touche pas une
// ligne d'`application/` ni de `domain/`.
package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/drivers/log"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/ports"
)

// Name est le nom du module, tel qu'il apparaît dans config/modules.yaml.
const Name = "notification"

// Pilotes disponibles.
const (
	driverLog = "log"
)

// Clés d'options lues par ce module, partagées avec le catalogue (ADR 014, #93).
const (
	// OptionBody décide si le corps du message est journalisé.
	OptionBody = "body"
)

// Valeurs admises par OptionBody.
//
// # Pourquoi deux mots plutôt qu'un booléen
//
// `body: false` répond à une question que le fichier ne pose pas — faux quoi ?
// `body: masked` et `body: logged` nomment les deux positions, donc se relisent
// sans aller chercher la documentation. Et une valeur mal orthographiée REFUSE
// le démarrage, là où un booléen mal typé se serait rabattu sur le défaut.
const (
	// BodyMasked tait le corps. C'est le défaut, et la position sûre.
	BodyMasked = "masked"

	// BodyLogged écrit le corps — développement uniquement.
	BodyLogged = "logged"
)

// Module expose les ports primaires.
//
// Une surface — HTTP, CLI, consommateur d'événements — ne reçoit QUE cette
// structure. Elle ne peut donc pas atteindre le fournisseur, ni contourner le
// cas d'usage.
type Module struct {
	Send ports.Send
}

// Deps porte les effets que le module ne fabrique pas lui-même.
type Deps struct {
	// Logger sert au pilote `log`. Il n'est PAS utilisé par les cas d'usage :
	// `application/` ne journalise pas, il rend compte.
	Logger *slog.Logger
}

// Erreurs du module.
var (
	// ErrDisabled signale un appel à un module désactivé.
	ErrDisabled = errors.New("module notification désactivé")

	// ErrMissingDependency refuse un montage incomplet.
	ErrMissingDependency = errors.New("dépendance obligatoire absente")

	errUnknownDriver = errors.New("pilote notification inconnu")

	errUnknownBodyOption = errors.New("valeur inconnue pour l'option body")
)

// New construit le module selon le pilote demandé.
//
// Un pilote inconnu REFUSE le démarrage : une faute de frappe ne se résout
// jamais en « le pilote le plus proche ». Deny par défaut.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return Disabled(), nil
	}

	driver := cfg.Driver
	if driver == "" {
		driver = driverLog
	}
	if driver != driverLog {
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, driver)
	}
	if deps.Logger == nil {
		return Module{}, fmt.Errorf("%w: Logger", ErrMissingDependency)
	}

	deliver, err := logDriver(cfg, deps.Logger)
	if err != nil {
		return Module{}, err
	}
	return Module{Send: application.NewSend(application.Deps{Deliver: deliver})}, nil
}

// logDriver choisit entre les deux constructeurs du pilote `log`.
//
// Une valeur inconnue REFUSE le démarrage plutôt que de se rabattre sur le
// défaut. Le repli serait pourtant tentant — il est « sûr », puisqu'il tait le
// corps — mais il ferait croire à quelqu'un qui a écrit `body: logué` que le
// corps sera écrit, et il chercherait ailleurs pendant une heure.
func logDriver(cfg config.Module, logger *slog.Logger) (ports.Deliver, error) {
	body, err := cfg.StringOption(OptionBody, BodyMasked)
	if err != nil {
		return nil, fmt.Errorf("modules.%s.%w", Name, err)
	}

	switch body {
	case BodyMasked:
		return log.New(logger), nil
	case BodyLogged:
		return log.NewIncludingBody(logger), nil
	default:
		return nil, fmt.Errorf(
			"%w: %q — attendu %q ou %q", errUnknownBodyOption, body, BodyMasked, BodyLogged)
	}
}

// Disabled rend un module qui refuse à l'appel.
//
// Il se monte toujours : c'est ce qui permet à un consommateur d'événements
// d'exister et de rendre compte clairement, plutôt que de faire échouer le
// démarrage entier pour un module que personne n'a activé.
//
// ⚠️ Il REFUSE, il n'avale pas. Un module de notification désactivé qui rendrait
// `nil` ferait compter chaque message comme envoyé — et le défaut ne se verrait
// qu'au premier client affirmant n'avoir jamais reçu son courriel, c'est-à-dire
// des semaines plus tard, sans aucune trace.
func Disabled() Module {
	return Module{
		Send: func(context.Context, domain.Message) error { return ErrDisabled },
	}
}
