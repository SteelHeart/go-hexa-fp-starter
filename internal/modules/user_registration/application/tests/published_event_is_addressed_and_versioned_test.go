package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestPublishedEventIsAddressedAndVersioned : l'événement part avec son nom versionné
// et l'identifiant de son agrégat.
//
// L'identifiant d'agrégat n'est pas décoratif : c'est lui qui permet de retrouver
// tous les événements d'un utilisateur lors d'un incident, et de rejouer une
// séquence dans le bon ordre. Sans lui, l'outbox n'est qu'un tas de messages.
//
// Le contenu est vérifié aussi : le condensé de mot de passe n'a rien à faire dans
// un message qui traversera un courtier.
func TestPublishedEventIsAddressedAndVersioned(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	_ = utilisateurDe(t, inscrit(depsNominales(observe)))

	if observe.typeEvent != domain.EventUserRegistered {
		t.Errorf("type d'événement = %q, attendu %q", observe.typeEvent, domain.EventUserRegistered)
	}
	if observe.agregat != identifiant.String() {
		t.Errorf("identifiant d'agrégat = %q, attendu %q", observe.agregat, identifiant)
	}

	encoded, err := json.Marshal(observe.evenement)
	if err != nil {
		t.Fatalf("l'événement doit être sérialisable: %v", err)
	}
	if strings.Contains(string(encoded), condense) {
		t.Errorf("l'événement transporte le condensé: %s", encoded)
	}
	if !strings.Contains(string(encoded), adresseValide) {
		t.Error("l'événement doit porter l'adresse : le consommateur en a besoin")
	}
}
