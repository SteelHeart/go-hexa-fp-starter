package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// Tests des identifiants NON EXPORTÉS du composition root du dépileur.
//
// `{paquet}/internal_test.go` et non `{paquet}/tests/` : `rules/tests.md` réserve
// le second à l'API publique d'un paquet, et Go INTERDIT d'importer un paquet
// `main`. Un test en boîte noire de ce binaire est donc impossible — c'est une
// contrainte du langage, pas une préférence.
//
// Ce qui est ÉPROUVÉ ici : les décisions de câblage, celles qui n'appartiennent à
// aucun module et qu'aucune règle d'architecture ne surveille. Le reste — le
// rejeu, la trace, l'acheminement — est éprouvé dans les paquets concernés.

// modulesConfig forge la configuration de deux modules.
func modulesConfig(notifOn, idemOn bool) config.Config {
	return config.Config{Modules: config.Modules{
		notification.Name: {Enabled: notifOn, Driver: "log"},
		idempotency.Name:  {Enabled: idemOn, Driver: "memory"},
	}}
}

// journalDeTest capte ce que le câblage écrit.
func journalDeTest() (*bytes.Buffer, *slog.Logger) {
	buffer := &bytes.Buffer{}
	return buffer, slog.New(slog.NewTextHandler(buffer, nil))
}

// brokerDeTest monte le relais en mémoire.
func brokerDeTest(t *testing.T, logger *slog.Logger) messaging.Broker {
	t.Helper()

	broker, err := messaging.New(config.Messaging{Driver: "inproc"}, logger)
	if err != nil {
		t.Fatalf("relais de test: %v", err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	return broker
}

// TestConsumingWithoutIdempotencyIsRefused garde contre le rejeu d'effets.
//
// # Pourquoi un refus et non un avertissement
//
// Tous les transports d'ici sont « au moins une fois » : la même enveloppe
// arrive deux fois dès qu'un accusé de réception se perd, ce qui est banal.
// S'abonner sans idempotence enverrait donc le courriel de bienvenue en double,
// et le jour où un consommateur débitera une carte, ce sera le débit.
//
// Un avertissement au démarrage est lu une fois puis jamais. Le refus, lui,
// nomme ce qu'il faut changer et se répare en une ligne de configuration.
func TestConsumingWithoutIdempotencyIsRefused(t *testing.T) {
	t.Parallel()

	_, logger := journalDeTest()
	err := abonner(modulesConfig(true, false), brokerDeTest(t, logger), logger)

	if !errors.Is(err, errIdempotenceRequise) {
		t.Fatalf("attendu errIdempotenceRequise, obtenu %v", err)
	}
}

// TestADisabledNotificationSaysSoRatherThanStayingSilent nomme la conséquence.
//
// Un module éteint est un état configuré, pas une panne : le démarrage ne doit
// pas échouer. Mais le SILENCE serait pire que l'inaction — un dépileur qui
// publie vers personne ressemble en tout point à un dépileur qui fonctionne :
// les messages sont marqués publiés, l'outbox se vide, les métriques sont
// vertes, et aucun effet n'a lieu.
//
// Le journal doit donc nommer la CONSÉQUENCE, pas seulement l'état.
func TestADisabledNotificationSaysSoRatherThanStayingSilent(t *testing.T) {
	t.Parallel()

	buffer, logger := journalDeTest()
	// L'idempotence est désactivée elle aussi : sans abonnement, elle n'est pas
	// requise. Le test constate donc AUSSI que le refus ne se déclenche pas à
	// tort — un garde trop large est un garde qu'on finit par retirer.
	if err := abonner(modulesConfig(false, false), brokerDeTest(t, logger), logger); err != nil {
		t.Fatalf("un module éteint ne doit pas faire échouer le démarrage : %v", err)
	}

	trace := buffer.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Fatalf("le journal doit nommer l'événement sans consommateur : %s", trace)
	}
	if !strings.Contains(trace, "personne n'y réagit") {
		t.Fatalf("le journal doit nommer la conséquence, pas seulement l'état : %s", trace)
	}
}

// TestTheWelcomeHandlerReadsThePublishedContract garde le langage publié.
//
// La charge utile est décodée en `contract.UserRegisteredV1` — des types
// primitifs, lisibles par un consommateur écrit dans un autre langage. Décoder
// vers un type du domaine de `user_registration` recréerait le couplage que le
// langage publié sert à éviter.
//
// Le test passe par une enveloppe RÉELLE, sérialisée comme le producteur la
// sérialise : c'est le seul moyen d'attraper un nom de champ JSON qui aurait
// dérivé d'un côté sans l'autre.
func TestTheWelcomeHandlerReadsThePublishedContract(t *testing.T) {
	t.Parallel()

	_, logger := journalDeTest()
	notif, err := notification.New(
		config.Module{Enabled: true, Driver: "log"}, notification.Deps{Logger: logger})
	if err != nil {
		t.Fatalf("module notification: %v", err)
	}

	charge, err := json.Marshal(contract.UserRegisteredV1{
		UserID: "compte-1", Email: "Alice@Example.COM",
	})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}

	err = bienvenue(notif)(context.Background(), messaging.Envelope{
		ID: "evt-1", Type: contract.EventUserRegisteredV1, Payload: charge,
	})
	if err != nil {
		t.Fatalf("le courriel de bienvenue doit partir : %v", err)
	}
}

// TestAnUnreadablePayloadIsReportedNotSwallowed interdit l'oubli silencieux.
//
// Une charge utile illisible remonte comme une erreur. L'avaler ferait acquitter
// l'enveloppe, donc perdre l'événement définitivement — et c'est au dépileur
// d'abandonner après N tentatives, pas à ce gestionnaire de décider seul
// d'oublier.
//
// Une adresse invalide suit le même chemin : elle vient du producteur, donc elle
// signale un défaut chez lui, pas une panne de transport.
func TestAnUnreadablePayloadIsReportedNotSwallowed(t *testing.T) {
	t.Parallel()

	_, logger := journalDeTest()
	notif, err := notification.New(
		config.Module{Enabled: true, Driver: "log"}, notification.Deps{Logger: logger})
	if err != nil {
		t.Fatalf("module notification: %v", err)
	}

	adresseVide, err := json.Marshal(contract.UserRegisteredV1{UserID: "compte-1"})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}

	cases := map[string][]byte{
		"charge illisible": []byte("{pas du json"),
		"charge vide":      nil,
		"adresse absente":  adresseVide,
	}
	for name, charge := range cases {
		err := bienvenue(notif)(context.Background(), messaging.Envelope{
			ID: "evt-1", Type: contract.EventUserRegisteredV1, Payload: charge,
		})
		if err == nil {
			t.Errorf("%s : l'erreur doit remonter, pas être avalée", name)
		}
	}
}
