// Package modulebus choisit COMMENT un module en appelle un autre, par
// configuration et sans changement de code.
//
// Trois modes, un par cycle de vie de déploiement :
//
//	inproc   appel de fonction direct — même binaire (défaut, le moins coûteux)
//	http     appel réseau au module distant — déploiements séparés
//	event    dépôt d'un événement — asynchrone, sans réponse
//
// Le passage de l'un à l'autre est une variable d'environnement :
//
//	MODULE_TRANSPORT_DEFAULT=inproc
//	MODULE_TRANSPORT=user_registration:http,billing:event
//	MODULE_BASE_URL=user_registration:http://user-registration:8080
//
// Ce paquet ne connaît AUCUN module en particulier : il manipule des types
// génériques et des contrats publiés (ADR 010).
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

// Route décrit l'exposition HTTP d'une capacité.
type Route struct {
	Method string
	Path   string
}

// Caller est une capacité appelable. Types primitifs ou contrats publiés
// uniquement : jamais un type du domaine d'un module.
type Caller[I any, O any] = func(ctx context.Context, in I) (O, error)

// Erreurs de résolution. Elles refusent le démarrage plutôt que de se rabattre
// silencieusement sur un mode qui n'était pas demandé — deny par défaut.
var (
	ErrNoBaseURL   = errors.New("aucune adresse configurée pour ce module")
	ErrUnknownMode = errors.New("mode de communication inter-module inconnu")
	ErrDisabled    = errors.New("communication désactivée pour ce module")
)

// Bus résout les capacités selon la configuration.
type Bus struct {
	cfg       config.Interop
	client    *http.Client
	publisher messaging.Publisher
}

// New construit le bus.
func New(cfg config.Interop, publisher messaging.Publisher) *Bus {
	return &Bus{
		cfg:       cfg,
		client:    &http.Client{Timeout: cfg.CallTimeout.Duration()},
		publisher: publisher,
	}
}

// Mode expose le mode retenu pour un module, à des fins de journalisation.
func (b *Bus) Mode(module string) Mode { return Mode(b.cfg.TransportFor(module)) }

// Resolve retourne l'appelable correspondant au mode configuré.
//
// `local` est l'implémentation en processus : elle n'est utilisée qu'en mode
// inproc, mais elle est toujours passée, ce qui garantit que le module local
// reste compilable et testable indépendamment du mode.
func Resolve[I, O any](
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
// Le corps d'erreur n'est PAS interprété : un module appelant n'a pas à
// connaître la taxonomie d'erreurs interne d'un autre. Il obtient le statut et
// le corps brut, et traduit lui-même.
func httpCaller[I, O any](client *http.Client, baseURL string, route Route) Caller[I, O] {
	endpoint := strings.TrimSuffix(baseURL, "/") + route.Path
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		body, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("sérialisation de la requête inter-module: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, route.Method, endpoint, bytes.NewReader(body))
		if err != nil {
			return zero, fmt.Errorf("construction de la requête inter-module: %w", err)
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
			return zero, fmt.Errorf("lecture de la réponse inter-module: %w", err)
		}
		return out, nil
	}
}

// eventCaller dépose un événement et retourne la valeur zéro.
//
// ⚠️ Mode ASYNCHRONE : il n'y a pas de réponse. Ne l'activer que pour une
// capacité dont l'appelant ignore le résultat. Le choix est explicite dans la
// configuration, donc auditable.
func eventCaller[I, O any](publisher messaging.Publisher, eventType string) Caller[I, O] {
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		payload, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("sérialisation de l'événement inter-module: %w", err)
		}
		env := messaging.Envelope{
			ID:         uuid.NewString(),
			Type:       eventType,
			Payload:    payload,
			OccurredAt: time.Now().UTC(),
		}
		if err := publisher(ctx, env); err != nil {
			return zero, fmt.Errorf("publication de l'événement inter-module: %w", err)
		}
		return zero, nil
	}
}
