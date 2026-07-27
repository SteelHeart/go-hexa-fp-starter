package tests

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestAnEventWithoutConsumerIsSignalledNotFailed : signalé, pas mis en échec.
//
// # Les deux défauts opposés que ce test attrape
//
// C'est un équilibre, et les deux côtés coûtent cher :
//
//  1. RENDRE UNE ERREUR serait faux. Le dépileur de l'outbox interpréterait
//     « personne n'écoute » comme « publication échouée » : il rejouerait, avec
//     un recul croissant, puis ABANDONNERAIT un événement parfaitement valide.
//     Or l'absence de consommateur est un état NORMAL du socle — aujourd'hui
//     même, `user.registered.v1` n'a aucun abonné.
//  2. NE RIEN DIRE serait faux aussi. Un module de notification mal monté, ou
//     un type d'événement renommé d'un côté seulement, produit exactement la
//     même trace qu'un fonctionnement nominal : le courriel ne part pas, et
//     rien ne l'explique.
//
// La sortie correcte est donc : succès, ET un avertissement qui NOMME le type.
func TestAnEventWithoutConsumerIsSignalledNotFailed(t *testing.T) {
	t.Parallel()

	logger, logs := recordingLogger(slog.LevelWarn)
	bus := messaging.NewInproc(logger)

	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("un événement sans consommateur a rendu %v — "+
			"le dépileur le prendrait pour un échec et finirait par l'abandonner", err)
	}

	trace := logs.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Errorf("l'avertissement ne nomme pas le type d'événement, trace=%q — "+
			"sans le nom, il est inexploitable", trace)
	}
}
