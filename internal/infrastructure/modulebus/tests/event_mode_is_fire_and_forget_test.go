package tests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestEventModeIsFireAndForget : l'événement part, et il n'y a PAS de réponse.
//
// # Ce que ce test verrouille
//
// Le mode `event` rend la valeur zéro avec `nil`. C'est correct — il n'y a pas de
// réponse dans un envoi asynchrone — et c'est un piège : la même signature que
// les deux autres modes rend une absence de réponse indiscernable d'une réponse
// négative.
//
// Le test l'affirme donc explicitement, pour que cette propriété soit une
// DÉCISION lisible et non un effet de bord. La configuration seule décide du
// mode : l'activer sur une capacité dont l'appelant lit le résultat le ferait
// lire « refusé » à chaque appel, sans aucune erreur.
//
// ⚠️ NON-GARANTIE CONNUE, ici parce qu'un test est le seul endroit qu'on relit :
// l'enveloppe produite ne porte AUCUN AggregateID — le bus est générique, il ne
// sait pas quelle entité l'appel concerne. Conséquence sur un relais Kafka :
// la clé de partition est vide, donc aucun ordre n'est garanti entre deux
// événements de la même entité, et tous atterrissent sur la même partition. Pour
// une capacité en oubli-après-envoi c'est acceptable ; ça ne l'est pas pour une
// capacité dont l'ordre compte. À trancher avant d'ouvrir ce mode largement.
func TestEventModeIsFireAndForget(t *testing.T) {
	t.Parallel()

	var published []messaging.Envelope
	publisher := func(_ context.Context, env messaging.Envelope) error {
		published = append(published, env)
		return nil
	}

	var localCalls int
	call := resolve(t, interop("event", nil), publisher, localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-7"})
	if err != nil {
		t.Fatalf("dépôt de l'événement en échec: %v", err)
	}

	if got.Accepted {
		t.Errorf("réponse = %+v, attendu la valeur zéro : un envoi asynchrone n'a pas de réponse", got)
	}
	if localCalls != 0 {
		t.Errorf("l'implémentation locale a été appelée %d fois en mode event", localCalls)
	}
	if len(published) != 1 {
		t.Fatalf("%d événement(s) publié(s), attendu 1", len(published))
	}

	env := published[0]
	if env.Type != someEvent {
		t.Errorf("type publié = %q, attendu %q", env.Type, someEvent)
	}
	if env.ID == "" {
		t.Error("événement sans identifiant — un consommateur ne peut plus dédupliquer")
	}
	if env.OccurredAt.IsZero() {
		t.Error("événement sans horodatage")
	}
	var payload request
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("charge utile illisible: %v", err)
	}
	if payload.Ref != "r-7" {
		t.Errorf("charge utile = %+v, la requête n'a pas traversé", payload)
	}
}

// TestEventModeSurfacesAPublicationFailure : un dépôt raté n'est pas un succès.
//
// # Le défaut que ce test attrape
//
// C'est le point le plus délicat de ce mode. « Oubli-après-envoi » qualifie
// l'absence de RÉPONSE, pas l'absence de GARANTIE : si la publication échoue,
// l'appel doit échouer. Avaler l'erreur ferait croire l'appelant servi alors que
// rien n'est parti — et comme le mode ne rend jamais de réponse, il n'existe
// aucun autre signal. La capacité serait morte en silence, définitivement.
func TestEventModeSurfacesAPublicationFailure(t *testing.T) {
	t.Parallel()

	refused := errors.New("relais indisponible")
	publisher := func(context.Context, messaging.Envelope) error { return refused }

	var localCalls int
	call := resolve(t, interop("event", nil), publisher, localCaller(&localCalls))

	_, err := call(context.Background(), request{Ref: "r-7"})

	if err == nil {
		t.Fatal("une publication en échec a rendu nil — la capacité serait morte en silence")
	}
	if !errors.Is(err, refused) {
		t.Errorf("la cause d'origine est perdue: %v", err)
	}
}
