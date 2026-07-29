// Package modulebus chooses HOW one module calls another, by configuration and
// without a code change.
//
// Three modes, one per deployment lifecycle:
//
//	inproc   direct function call — same binary (default, the cheapest)
//	http     network call to the remote module — separate deployments
//	event    posting an event — asynchronous, no reply
//
// Switching from one to another is an environment variable:
//
//	MODULE_TRANSPORT_DEFAULT=inproc
//	MODULE_TRANSPORT=user_registration:http,billing:event
//	MODULE_BASE_URL=user_registration:http://user-registration:8080
//
// This package knows NO module in particular: it handles generic types and
// published contracts (ADR 010).
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

// Mode names an inter-module communication mode.
type Mode string

// The available modes.
const (
	ModeInproc   Mode = "inproc"
	ModeHTTP     Mode = "http"
	ModeEvent    Mode = "event"
	ModeDisabled Mode = "disabled"
)

// Route describes the HTTP exposure of a capability.
type Route struct {
	Method string
	Path   string
}

// Caller is a callable capability. Primitive types or published contracts
// only: never a type from a module's domain.
type Caller[I any, O any] = func(ctx context.Context, in I) (O, error)

// Resolution errors. They refuse start-up rather than silently falling back on
// a mode that was not asked for — deny by default.
var (
	ErrNoBaseURL   = errors.New("no address configured for this module")
	ErrUnknownMode = errors.New("unknown inter-module communication mode")
	ErrDisabled    = errors.New("communication disabled for this module")
)

// Bus resolves capabilities according to the configuration.
type Bus struct {
	cfg       config.Interop
	client    *http.Client
	publisher messaging.Publisher
}

// New builds the bus.
func New(cfg config.Interop, publisher messaging.Publisher) *Bus {
	return &Bus{
		cfg:       cfg,
		client:    &http.Client{Timeout: cfg.CallTimeout.Duration()},
		publisher: publisher,
	}
}

// Mode exposes the mode selected for a module, for logging purposes.
func (b *Bus) Mode(module string) Mode { return Mode(b.cfg.TransportFor(module)) }

// Resolve returns the callable matching the configured mode.
//
// `local` is the in-process implementation: it is only used in inproc mode, but
// it is always passed, which guarantees that the local module stays compilable
// and testable independently of the mode.
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
		return nil, fmt.Errorf("%w: %q for %s", ErrUnknownMode, mode, module)
	}
}

// httpCaller calls the remote module.
//
// The error body is NOT interpreted: a calling module has no business knowing
// another one's internal error taxonomy. It gets the status and the raw body,
// and translates it itself.
func httpCaller[I, O any](client *http.Client, baseURL string, route Route) Caller[I, O] {
	endpoint := strings.TrimSuffix(baseURL, "/") + route.Path
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		body, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("serialising the inter-module request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, route.Method, endpoint, bytes.NewReader(body))
		if err != nil {
			return zero, fmt.Errorf("building the inter-module request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return zero, fmt.Errorf("inter-module call %s: %w", endpoint, err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode >= http.StatusBadRequest {
			return zero, fmt.Errorf("inter-module call %s: status %d", endpoint, resp.StatusCode)
		}
		var out O
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return zero, fmt.Errorf("reading the inter-module reply: %w", err)
		}
		return out, nil
	}
}

// eventCaller posts an event and returns the zero value.
//
// ⚠️ ASYNCHRONOUS mode: there is no reply. Only enable it for a capability
// whose caller ignores the result. The choice is explicit in the configuration,
// hence auditable.
func eventCaller[I, O any](publisher messaging.Publisher, eventType string) Caller[I, O] {
	return func(ctx context.Context, in I) (O, error) {
		var zero O
		payload, err := json.Marshal(in)
		if err != nil {
			return zero, fmt.Errorf("serialising the inter-module event: %w", err)
		}
		env := messaging.Envelope{
			ID:         uuid.NewString(),
			Type:       eventType,
			Payload:    payload,
			OccurredAt: time.Now().UTC(),
		}
		if err := publisher(ctx, env); err != nil {
			return zero, fmt.Errorf("publishing the inter-module event: %w", err)
		}
		return zero, nil
	}
}
