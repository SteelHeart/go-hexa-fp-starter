// Package tests exerce le bus inter-module par son API PUBLIQUE.
//
// Ce paquet décide COMMENT un module en appelle un autre — appel direct, appel
// réseau, ou dépôt d'événement — par configuration seule. C'est donc le point où
// une erreur de configuration se transforme en appel qui part au mauvais endroit,
// ou qui ne part pas du tout.
//
// Les trois modes se testent SANS INFRASTRUCTURE : `inproc` est une closure,
// `http` se sert d'un `httptest.Server` en mémoire, `event` d'un publieur qui
// est lui-même une closure. C'est la promesse du socle, appliquée à son propre
// outillage de test.
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// Le module et la capacité fictifs de ces tests. Aucun module réel n'est nommé :
// le bus ne connaît AUCUN module en particulier, et le test doit refléter ça.
const (
	someModule = "some_module"
	someEvent  = "some_module.thing_done.v1"
)

// request et reply sont les types traversant le bus : des enregistrements
// sérialisables, jamais un type du domaine d'un module.
type request struct {
	Ref string `json:"ref"`
}

type reply struct {
	Accepted bool `json:"accepted"`
}

// interop construit une configuration d'interopérabilité.
func interop(mode string, baseURLs map[string]string) config.Interop {
	return config.Interop{
		DefaultTransport: mode,
		CallTimeout:      config.Duration(2 * time.Second),
		Transports:       map[string]string{},
		BaseURLs:         baseURLs,
	}
}

// route est l'exposition HTTP de la capacité fictive.
func route() modulebus.Route {
	return modulebus.Route{Method: "POST", Path: "/v1/things"}
}

// localCaller est l'implémentation en processus. Elle est TOUJOURS passée à
// Resolve, quel que soit le mode : c'est ce qui garantit que le module local
// reste compilable et testable indépendamment du transport retenu.
func localCaller(called *int) modulebus.Caller[request, reply] {
	return func(_ context.Context, in request) (reply, error) {
		*called++
		return reply{Accepted: in.Ref != ""}, nil
	}
}

// resolve monte le bus et résout la capacité, ou fait échouer le test.
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
		t.Fatalf("résolution en échec: %v", err)
	}
	return call
}

// noPublisher refuse toute publication : il est passé aux tests des modes qui ne
// doivent PAS publier. Un mode `http` qui déposerait un événement au passage
// serait sinon indétectable.
func noPublisher(t *testing.T) messaging.Publisher {
	t.Helper()

	return func(_ context.Context, env messaging.Envelope) error {
		t.Errorf("publication inattendue d'un événement %q", env.Type)
		return nil
	}
}
