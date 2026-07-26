// Package tests exerce le relais d'événements par son API PUBLIQUE.
//
// Ce que ces tests NE couvrent pas, et il faut le dire : les relais `kafka` et
// `rabbitmq` ouvrent une connexion réseau dès leur construction. Aucun broker
// n'existe ici, donc ils restent ÉCRITS, NON PROUVÉS — c'est le niveau
// `-tags=integration` qui les exercera, en CI.
//
// Tout le reste — le choix du relais, le bus en mémoire, le relais muet, le
// décorateur de réessai, l'enveloppe — se teste sans aucune infrastructure, et
// c'est exactement ce que le socle promet.
package tests

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// quietLogger rend un journal muet, pour les tests qui ne l'observent pas.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

// recordingLogger rend un journal ET son tampon : certains tests vérifient
// qu'une situation anormale a bien été SIGNALÉE, pas seulement tolérée.
func recordingLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})
	return slog.New(handler), buf
}

// relayConfig construit une configuration minimale pour un relais donné.
func relayConfig(driver string) config.Messaging {
	return config.Messaging{
		Driver:         driver,
		TopicPrefix:    "hexa",
		ConsumerGroup:  "tests",
		PublishTimeout: config.Duration(time.Second),
	}
}

// envelope construit une enveloppe de test au type donné.
func envelope(eventType string) messaging.Envelope {
	return messaging.Envelope{
		ID:          "01890000-0000-7000-8000-000000000000",
		Type:        eventType,
		AggregateID: "agg-1",
		Payload:     []byte(`{"k":"v"}`),
		OccurredAt:  time.Date(2026, time.July, 26, 10, 0, 0, 0, time.UTC),
	}
}

// mustBroker monte un relais, ou fait échouer le test.
func mustBroker(t *testing.T, driver string) messaging.Broker {
	t.Helper()

	broker, err := messaging.New(relayConfig(driver), quietLogger())
	if err != nil {
		t.Fatalf("montage du relais %q en échec: %v", driver, err)
	}
	return broker
}

// cancelled rend un contexte déjà annulé, pour réveiller un Run bloquant sans
// attendre.
func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
