package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
)

// TestADisabledModuleRefusesItNeverSwallows garde le pire silence possible.
//
// # Le défaut que ce test empêche
//
// Un module de notification désactivé qui rendrait `nil` ferait compter chaque
// message comme envoyé. Rien n'échouerait, aucun journal ne dirait rien, et le
// défaut ne se verrait qu'au premier client affirmant n'avoir jamais reçu son
// courriel — des semaines plus tard, sans aucune trace pour remonter.
//
// C'est la forme la plus coûteuse d'un défaut : celle qui ressemble en tout
// point au succès.
func TestADisabledModuleRefusesItNeverSwallows(t *testing.T) {
	t.Parallel()

	mod, err := notification.New(config.Module{Enabled: false}, notification.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé doit se monter : %v", err)
	}

	if err := mod.Send(context.Background(), message(t)); !errors.Is(err, notification.ErrDisabled) {
		t.Fatalf("attendu ErrDisabled, obtenu %v", err)
	}
}

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
//
// `smtp`, `mailjet` et `ses` figurent dans le catalogue d'intentions et ne sont
// PAS construits. Le refus doit être franc : un repli silencieux sur `log`
// ferait tourner une production qui écrit ses courriels dans un journal au lieu
// de les envoyer — et rien ne le signalerait.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	j := newJournal()
	for _, driver := range []string{"smtp", "mailjet", "ses", "sendgrid", "journal"} {
		_, err := notification.New(
			config.Module{Enabled: true, Driver: driver},
			notification.Deps{Logger: j.logger},
		)
		if err == nil {
			t.Errorf("le pilote %q n'est pas construit : il doit refuser le démarrage", driver)
		}
	}
}

// TestMissingLoggerRefusesStartup fait échouer le montage, pas la production.
//
// Sans ce refus, le premier message produirait une déréférence de pointeur nil —
// donc en production, et sur un chemin asynchrone où la panique du consommateur
// se confondrait avec un message mal formé.
func TestMissingLoggerRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := notification.New(config.Module{Enabled: true, Driver: "log"}, notification.Deps{})
	if !errors.Is(err, notification.ErrMissingDependency) {
		t.Fatalf("attendu ErrMissingDependency, obtenu %v", err)
	}
}

// TestEmptyDriverFallsBackToTheDefault : ne rien préciser prend `log`.
//
// C'est ce qui rend vraie la promesse « `hexa new` puis `go run` » sur la chaîne
// COMPLÈTE — inscription, outbox, relais, notification — sans serveur SMTP ni
// compte chez un fournisseur.
func TestEmptyDriverFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	j := newJournal()
	mod, err := notification.New(config.Module{Enabled: true}, notification.Deps{Logger: j.logger})
	if err != nil {
		t.Fatalf("sans pilote précisé, le défaut doit s'appliquer : %v", err)
	}
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("envoi sur le pilote par défaut : %v", err)
	}
	if j.texte() == "" {
		t.Fatal("le pilote par défaut doit être le pilote `log`")
	}
}

// TestAMessageNeverPrintsItsBody couvre les DEUX verbes de formatage.
//
// `%v` passe par `String()`, `%#v` par `GoString()`. Couvrir l'un laisse l'autre
// fuiter, et `%#v` est précisément ce qu'on écrit dans un journal de débogage —
// donc le jour d'un incident, donc le jour où les journaux partent chez un tiers.
//
// ⚠️ Le test formate DÉLIBÉRÉMENT plutôt que d'appeler `String()` : c'est le
// chemin de fuite réel qui est éprouvé. Appeler la méthode laisserait le test
// vert si quelqu'un retirait le `Stringer`.
func TestAMessageNeverPrintsItsBody(t *testing.T) {
	t.Parallel()

	msg := message(t)
	for _, rendu := range []string{
		fmt.Sprintf("%v", msg),  //nolint:gocritic // c'est le chemin de fuite testé
		fmt.Sprintf("%#v", msg), // par GoString — l'autre moitié du masque
		fmt.Sprint(msg),         //nolint:gocritic // idem, sans verbe
	} {
		if strings.Contains(rendu, secretInBody) {
			t.Fatalf("le corps fuite dans %q", rendu)
		}
		if strings.Contains(rendu, recipient) {
			t.Fatalf("l'adresse en clair fuite dans %q", rendu)
		}
		if !strings.Contains(rendu, "***") {
			t.Fatalf("le masque doit être visible dans %q", rendu)
		}
	}
}
