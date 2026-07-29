// Package notification is the composition root of the notification module.
//
// It is the ONLY file of the module that knows the drivers (ADR 012). The use
// cases only see function types: changing driver does not touch a line of
// `application/` nor of `domain/`.
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

// Name is the module name, as it appears in config/modules.yaml.
const Name = "notification"

// Available drivers.
const (
	driverLog = "log"
)

// Option keys read by this module, shared with the catalogue (ADR 014, #93).
const (
	// OptionBody decides whether the message body is logged.
	OptionBody = "body"
)

// Values admitted by OptionBody.
//
// # Why two words rather than a boolean
//
// `body: false` answers a question the file does not ask — false what?
// `body: masked` and `body: logged` name the two positions, hence read back
// without going to look up the documentation. And a misspelled value REFUSES
// startup, where a mistyped boolean would have fallen back to the default.
const (
	// BodyMasked keeps the body silent. It is the default, and the safe
	// position.
	BodyMasked = "masked"

	// BodyLogged writes the body — development only.
	BodyLogged = "logged"
)

// Module exposes the primary ports.
//
// A surface — HTTP, CLI, event consumer — receives ONLY this struct. It
// therefore cannot reach the provider, nor bypass the use case.
type Module struct {
	Send ports.Send
}

// Deps carries the effects the module does not build itself.
type Deps struct {
	// Logger serves the `log` driver. It is NOT used by the use cases:
	// `application/` does not log, it reports.
	Logger *slog.Logger
}

// Errors of the module.
var (
	// ErrDisabled signals a call to a disabled module.
	ErrDisabled = errors.New("notification module disabled")

	// ErrMissingDependency refuses an incomplete assembly.
	ErrMissingDependency = errors.New("mandatory dependency missing")

	errUnknownDriver = errors.New("unknown notification driver")

	errUnknownBodyOption = errors.New("unknown value for the body option")
)

// New builds the module according to the requested driver.
//
// An unknown driver REFUSES startup: a typo never resolves into "the closest
// driver". Deny by default.
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

// logDriver chooses between the two constructors of the `log` driver.
//
// An unknown value REFUSES startup rather than falling back to the default. The
// fallback would be tempting though — it is "safe", since it keeps the body
// silent — but it would make someone who wrote `body: logging` believe the body
// will be written, and they would search elsewhere for an hour.
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
			"%w: %q — expected %q or %q", errUnknownBodyOption, body, BodyMasked, BodyLogged)
	}
}

// Disabled returns a module that refuses on call.
//
// It always mounts: that is what lets an event consumer exist and report
// clearly, rather than failing the whole startup for a module nobody enabled.
//
// ⚠️ It REFUSES, it does not swallow. A disabled notification module returning
// `nil` would make every message count as sent — and the defect would only show
// at the first customer claiming they never received their email, that is to say
// weeks later, without any trace.
func Disabled() Module {
	return Module{
		Send: func(context.Context, domain.Message) error { return ErrDisabled },
	}
}
