package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestUnmarkableMessageIsReportedAsDuplicateRisk couvre le cas le plus grave et le
// moins évident du dépilage.
//
// Le message a été PUBLIÉ, mais la base est tombée avant qu'on puisse l'enregistrer.
// Il reste donc `pending` et repartira au prochain tour : le consommateur recevra
// un doublon.
//
// Ce n'est pas un défaut qu'on peut supprimer — publier et enregistrer sont deux
// systèmes, et aucune transaction ne les couvre ensemble. C'est exactement pourquoi
// la livraison est « au moins une fois » et pourquoi `ports.Handler` impose
// l'idempotence au consommateur. Ce qui EST exigible, c'est que le risque soit
// rapporté distinctement plutôt que confondu avec un échec de publication.
func TestUnmarkableMessageIsReportedAsDuplicateRisk(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	var published bool
	handle := func(context.Context, domain.Message) error { published = true; return nil }

	failingMarkDone := func(context.Context, domain.MessageID) error {
		return errors.New("base injoignable")
	}

	dispatcher := newDispatcher(t, application.Ports{
		Claim:      claimOnce(pending("m-1", 0)),
		MarkDone:   failingMarkDone,
		MarkFailed: observed.markFailed(),
		Handle:     handle,
		Report:     observed.report(),
		Now:        newDispatchClock().now,
	}, testPolicy())

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	if !published {
		t.Fatal("le message devait être publié avant l'échec d'enregistrement")
	}
	if len(observed.failed) != 0 {
		t.Error("un message PUBLIÉ ne doit pas être reprogrammé comme un échec de publication")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventResolveFailed {
		t.Errorf("événement = %q, attendu %q", got, domain.EventResolveFailed)
	}
}
