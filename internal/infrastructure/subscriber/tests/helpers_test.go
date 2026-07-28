// Package tests éprouve le branchement d'un consommateur sur les garanties du
// noyau.
//
// # Le module d'idempotence est le VRAI, pas un double
//
// Le pilote `memory` est celui qui tourne en développement, et ses 24 tests
// prouvent l'exclusivité sous concurrence. Le doubler ici ferait éprouver le
// double : le décorateur pourrait respecter un contrat que le vrai module ne
// tient pas, et le rejeu passerait quand même en production.
//
// Ce qui EST doublé, c'est le gestionnaire — parce que ce qu'on mesure est
// justement le nombre de fois où il est appelé.
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// eventType est le type d'événement utilisé par les tests.
const eventType = "user.registered.v1"

// newGuard monte le VRAI module d'idempotence sur son pilote par défaut.
func newGuard(t *testing.T) subscriber.Guard {
	t.Helper()

	mod, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memory", Options: map[string]any{"ttl": "1h"}},
		idempotency.Deps{Now: time.Now},
	)
	if err != nil {
		t.Fatalf("montage de l'idempotence: %v", err)
	}
	return subscriber.Guard{Reserve: mod.Reserve, Complete: mod.Complete, Release: mod.Release}
}

// envelope forge une enveloppe de transport.
func envelope(id string) messaging.Envelope {
	return messaging.Envelope{
		ID:          id,
		Type:        eventType,
		AggregateID: "compte-1",
		Payload:     []byte(`{"user_id":"compte-1","email":"alice@example.com"}`),
		OccurredAt:  time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC),
	}
}

// compteur retient combien de fois le gestionnaire a été appelé.
//
// C'est la seule chose que ces tests mesurent vraiment : « l'effet a-t-il eu
// lieu, et combien de fois ». Un compteur sûr en concurrence, parce qu'un des
// tests rejoue depuis plusieurs goroutines.
type compteur struct {
	mu      sync.Mutex
	appels  int
	erreur  error
	contexs []context.Context
}

// handler rend un gestionnaire qui compte ses appels.
func (c *compteur) handler() messaging.Handler {
	return func(ctx context.Context, _ messaging.Envelope) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.appels++
		c.contexs = append(c.contexs, ctx)
		return c.erreur
	}
}

// total rend le nombre d'appels observés.
func (c *compteur) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.appels
}

// dernierContexte rend le contexte reçu au dernier appel.
func (c *compteur) dernierContexte(t *testing.T) context.Context {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.contexs) == 0 {
		t.Fatal("le gestionnaire n'a jamais été appelé : le test n'observe rien")
	}
	return c.contexs[len(c.contexs)-1]
}

// echouant rend un compteur dont le gestionnaire échoue toujours.
func echouant(err error) *compteur { return &compteur{erreur: err} }
