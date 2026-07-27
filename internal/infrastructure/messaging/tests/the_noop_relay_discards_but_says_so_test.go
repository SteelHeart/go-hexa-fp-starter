package tests

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheNoopRelayDiscardsButSaysSo : le relais muet jette, et le DIT.
//
// # Le défaut que ce test attrape
//
// Un relais qui jette les événements SANS RIEN DIRE est indiscernable d'un
// relais en panne. C'est le seul pilote du socle dont le comportement nominal
// est la perte : il doit donc être le plus bavard, pas le plus silencieux.
//
// La trace est en Debug volontairement — en production le relais `noop` n'a rien
// à faire, et l'exiger en Info noierait les journaux d'un environnement de test.
// Mais elle doit NOMMER le type d'événement : « publication ignorée » sans le
// nom n'aide personne à comprendre pourquoi son courriel n'est pas parti.
func TestTheNoopRelayDiscardsButSaysSo(t *testing.T) {
	t.Parallel()

	logger, logs := recordingLogger(slog.LevelDebug)

	broker, err := messaging.New(relayConfig(string(messaging.DriverNoop)), logger)
	if err != nil {
		t.Fatalf("montage du relais noop en échec: %v", err)
	}

	// Un consommateur enregistré sur le relais muet ne doit RIEN recevoir :
	// c'est ce qui le rend utilisable en migration sans effet de bord.
	var received int
	broker.Consume.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		received++
		return nil
	})

	if err := broker.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("le relais muet a rendu une erreur: %v", err)
	}
	if received != 0 {
		t.Errorf("le relais muet a livré %d événement(s) — il ne doit rien livrer", received)
	}

	trace := logs.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Errorf("la trace ne nomme pas le type ignoré, trace=%q", trace)
	}
}

// TestTheNoopConsumerReturnsOnShutdown : Run rend la main à l'annulation.
//
// Un consommateur qui ne surveillerait pas le contexte bloquerait l'arrêt du
// worker jusqu'au SIGKILL — y compris pour le pilote qui, par définition, n'a
// rien à terminer.
func TestTheNoopConsumerReturnsOnShutdown(t *testing.T) {
	t.Parallel()

	broker := mustBroker(t, string(messaging.DriverNoop))

	done := make(chan error, 1)
	go func() { done <- broker.Consume.Run(cancelled()) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run a rendu %v à l'annulation, attendu nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run n'a pas rendu la main sur un contexte annulé — l'arrêt du worker bloquerait")
	}
}
