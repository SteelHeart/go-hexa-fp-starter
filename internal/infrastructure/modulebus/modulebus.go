// Package modulebus choisit COMMENT un module en appelle un autre, par
// configuration et sans changement de code.
//
// Trois modes, un par cycle de vie de dÃ©ploiement :
//
//	inproc   appel de fonction direct â€” mÃªme binaire (dÃ©faut, le moins coÃ»teux)
//	http     appel rÃ©seau au module distant â€” dÃ©ploiements sÃ©parÃ©s
//	event    dÃ©pÃ´t d'un Ã©vÃ©nement â€” asynchrone, sans rÃ©ponse
//
// Le passage de l'un Ã  l'autre est une variable d'environnement :
//
//	MODULE_TRANSPORT_DEFAULT=inproc
//	MODULE_TRANSPORT=user_registration:http,billing:event
//	MODULE_BASE_URL=user_registration:http://user-registration:8080
//
// Ce paquet ne connaÃ®t AUCUN module en particulier : il manipule des types
// gÃ©nÃ©riques et des contrats publiÃ©s (ADR 010).
package modulebus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// Mode nomme un mode de communication inter-module.
type Mode string

// Les modes disponibles.
const (
	ModeInproc   Mode = "inproc"
	ModeHTTP     Mode = "http"
	ModeEvent    Mode = "event"
	ModeDisabled Mode = "disabled"
)

// Route dÃ©crit l'exposition HTTP d'une capacitÃ©.
type Route struct {
	Method string
	Path   string
}

// Caller est une capacitÃ© appelable. Types primitifs ou contrats publiÃ©s
// uniquement : jamais un type du domaine d'un module.
type Caller[I any, O any] = func(ctx context.Context, in I) (O, error)

// Erreurs de rÃ©solution. Elles refusent le dÃ©marrage plutÃ´t que de se rabattre
// silencieusement sur un mode qui n'Ã©tait pas demandÃ© â€” deny par dÃ©faut.
var (
	ErrNoBaseURL   = errors.New("aucune adresse configurÃ©e pour ce module")
	ErrUnknownMode = errors.New("mode de communication inter-module inconnu")
	ErrDisabled    = errors.New("communication dÃ©sactivÃ©e pour ce module")
)

// Bus rÃ©sout les capacitÃ©s selon la configuration.
type Bus struct {
	cfg       config.Modules
	client    *http.Client
	publisher messaging.Publisher
}

// New construit le bus.
func New(cfg config.Modules, publisher messaging.Publisher) *Bus {
	return &Bus{
		cfg:       cfg,
		client:    &http.Client{Timeout: cfg.CallTimeout},
		publisher: publisher,
	}
}

// Mode expose le mode retenu pour un module, Ã  des fins de journalisation.
func (b *Bus) Mode(module string) Mode { return Mode(b.cfg.TransportFor(module)) }

// Resolve retourne l'appelable correspondant au mode configurÃ©.
//
// `local` est l'implÃ©mentation en processus : elle n'est utilisÃ©e qu'en mode
// inproc, mais elle est toujours passÃ©e, ce qui garantit que le module local
// reste compilable et testable indÃ©pendamment du mode.
func Resolve[I any, O any](
	bus *Bus,
	module string,
	route Route,
	eventType string,
	local Caller[I, O],
) (Caller[I, O], error) {
	switch mode := Mode(bus.cfg.TransportFor(module)); mode {
	case ModeInproc:
		return local, nil
	case ModeHTTP:
		baseURL, found := bus.cfg.BaseURLs[module]
		if !found || baseURL == "" {
			return nil, fmt.Errorf("%w: %s", ErrNoBaseURL, module)
		}
		return httpCaller[I, O](bus.client, baseURL, route), nil
	case ModeEvent:
		return eventCaller[I, O](bus.publisher, eventType), nil
	case ModeDisabled:
		return func(context.Context, I) (O, error) {
			var zero O
			return zero, fmt.Errorf("%w: %s", ErrDisabled, module)
		}, nil
	default:
		return nil, fmt.Errorf("%w: %q pour %s", ErrUnknownMode, mode, module)
	}
}

// httpCaller appelle le module distant.
//
// Le corps d'erreur n'est PAS interprÃ©tÃ© : un module appelant n'a pas Ã 
// connaÃ®tre la taxonomie d'erreurs interne d'un autre. Il obtient le statut et
// le corps brut, et traduit lui-mÃªme.
func httpCaller[I any, O any](client *http.Client, baseURL string, route Route) Caller[I, O] {
	endpoint := strings.TrimSuffix(baseURL, "/") + route.Path
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		body, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("sÃ©rialisation de la requÃªte inter-module: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, route.Method, endpoint, bytes.NewReader(body))
		if err != nil {
			return zero, fmt.Errorf("construction de la requÃªte inter-module: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return zero, fmt.Errorf("appel inter-module %s: %w", endpoint, err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= http.StatusBadRequest {
			return zero, fmt.Errorf("appel inter-module %s: statut %d", endpoint, resp.StatusCode)
		}
		var out O
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return zero, fmt.Errorf("lecture de la rÃ©ponse inter-module: %w", err)
		}
		return out, nil
	}
}

// eventCaller dÃ©pose un Ã©vÃ©nement et retourne la valeur zÃ©ro.
//
// âš ï¸ Mode ASYNCHRONE : il n'y a pas de rÃ©ponse. Ne l'activer que pour une
// capacitÃ© dont l'appelant ignore le rÃ©sultat. Le choix est explicite dans la
// configuration, donc auditable.
func eventCaller[I any, O any](publisher messaging.Publisher, eventType string) Caller[I, O] {
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		payload, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("sÃ©rialisation de l'Ã©vÃ©nement inter-module: %w", err)
		}
		env := messaging.Envelope{
			ID:         uuid.NewString(),
			Type:       eventType,
			Payload:    payload,
			OccurredAt: time.Now().UTC(),
		}
		if err := publisher(ctx, env); err != nil {
			return zero, fmt.Errorf("publication de l'Ã©vÃ©nement inter-module: %w", err)
		}
		return zero, nil
	}
}
