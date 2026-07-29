// Package tests exercises the inter-module bus through its PUBLIC API.
//
// This package decides HOW one module calls another — direct call, network
// call, or event posting — by configuration alone. It is therefore the point
// where a configuration mistake turns into a call that goes to the wrong place,
// or that does not go at all.
//
// The three modes are tested WITHOUT INFRASTRUCTURE: `inproc` is a closure,
// `http` uses an in-memory `httptest.Server`, `event` a publisher that is
// itself a closure. That is the starter's promise, applied to its own test
// tooling.
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// The fictitious module and capability of these tests. No real module is named:
// the bus knows NO module in particular, and the test must reflect that.
const (
	someModule = "some_module"
	someEvent  = "some_module.thing_done.v1"
)

// request and reply are the types crossing the bus: serialisable records, never
// a type from a module's domain.
type request struct {
	Ref string `json:"ref"`
}

type reply struct {
	Accepted bool `json:"accepted"`
}

// interop builds an interoperability configuration.
func interop(mode string, baseURLs map[string]string) config.Interop {
	return config.Interop{
		DefaultTransport: mode,
		CallTimeout:      config.Duration(2 * time.Second),
		Transports:       map[string]string{},
		BaseURLs:         baseURLs,
	}
}

// route is the HTTP exposure of the fictitious capability.
func route() modulebus.Route {
	return modulebus.Route{Method: "POST", Path: "/v1/things"}
}

// localCaller is the in-process implementation. It is ALWAYS passed to Resolve,
// whatever the mode: that is what guarantees the local module stays compilable
// and testable independently of the transport selected.
func localCaller(called *int) modulebus.Caller[request, reply] {
	return func(_ context.Context, in request) (reply, error) {
		*called++
		return reply{Accepted: in.Ref != ""}, nil
	}
}

// resolve mounts the bus and resolves the capability, or fails the test.
func resolve(
	t *testing.T,
	cfg config.Interop,
	publisher messaging.Publisher,
	local modulebus.Caller[request, reply],
) modulebus.Caller[request, reply] {
	t.Helper()

	call, err := modulebus.Resolve(
		modulebus.New(cfg, publisher), someModule, route(), someEvent, local)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}
	return call
}

// noPublisher refuses any publication: it is passed to the tests of the modes
// that must NOT publish. An `http` mode that posted an event along the way
// would otherwise be undetectable.
func noPublisher(t *testing.T) messaging.Publisher {
	t.Helper()

	return func(_ context.Context, env messaging.Envelope) error {
		t.Errorf("unexpected publication of an event %q", env.Type)
		return nil
	}
}
