package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

// TestUnsharedDriverRefusesASeparateDispatcher : un pilote non partagé refuse un
// dépileur lancé dans un AUTRE processus.
//
// # Le défaut que ce refus empêche
//
// Le pilote `memory` vit dans le processus. Un dépileur lancé comme binaire
// séparé interrogerait donc son propre magasin — vide — pendant que les
// événements écrits par le serveur resteraient dans la mémoire du serveur.
//
// Il tournerait sans rien publier ET sans aucune erreur : journaux propres,
// sonde verte, processus vivant. Le défaut ne se découvrirait que le jour où
// quelqu'un demande pourquoi un client n'a jamais reçu son courriel.
//
// Deny par défaut jusque dans la valeur inconnue : un pilote qu'on ne connaît
// pas est réputé NON partagé. Se tromper dans ce sens fait échouer un
// démarrage ; se tromper dans l'autre fait tourner un dépileur à vide pendant
// des mois.
func TestUnsharedDriverRefusesASeparateDispatcher(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"memory, non partagé entre processus": "memory",
		"pilote absent":                       "",
		"pilote inconnu":                      "cassandra",
	}
	for name, driver := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := outbox.RequireSharedDriver(driver)
			if !errors.Is(err, outbox.ErrNotShared) {
				t.Fatalf("RequireSharedDriver(%q) = %v, attendu un refus explicite", driver, err)
			}
			if outbox.SharedAcrossProcesses(driver) {
				t.Errorf("SharedAcrossProcesses(%q) = true, attendu false", driver)
			}
		})
	}

	if err := outbox.RequireSharedDriver("postgres"); err != nil {
		t.Errorf("le pilote postgres est partagé et doit être accepté, reçu: %v", err)
	}
	if !outbox.SharedAcrossProcesses("postgres") {
		t.Error("SharedAcrossProcesses(\"postgres\") doit être true")
	}
}
