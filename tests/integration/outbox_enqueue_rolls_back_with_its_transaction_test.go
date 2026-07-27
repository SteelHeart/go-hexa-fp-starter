//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/database"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestOutboxEnqueueRollsBackWithItsTransaction vérifie la raison d'être de
// l'outbox transactionnelle (ADR 006).
//
// La promesse s'énonce en une phrase et ne se vérifie pas sans base :
// l'événement et la donnée métier vivent ou meurent ENSEMBLE. Si l'un des deux
// pouvait survivre seul, l'outbox n'apporterait rien qu'une table de plus.
//
// Les deux modes de défaillance évités sont opposés, et coûtent tous les deux :
//
//   - l'écriture métier échoue, l'événement part → un consommateur agit sur un
//     fait qui n'a jamais eu lieu ;
//   - l'écriture métier réussit, l'événement se perd → personne n'agit, et rien
//     ne le signale.
//
// Le second est le pire : il ne produit aucune erreur.
//
// Le test passe par `RunInTx`, le SEUL chemin qui ouvre une transaction dans ce
// socle — il n'existe volontairement aucun moyen exporté d'en poser une dans un
// contexte. Un `Result` en `Err` déclenche l'annulation : l'erreur métier est
// transactionnellement significative, et c'est précisément ce qu'on éprouve.
func TestOutboxEnqueueRollsBackWithItsTransaction(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	eventType := unique(t, "integration.rollback")
	unitOfWork := database.RunInTx[int, string](p)

	// Une unité de travail qui écrit puis ÉCHOUE.
	res := unitOfWork(ctx, func(txCtx context.Context) result.Result[int, string] {
		if _, err := store.Enqueue(txCtx, domain.NewMessage{
			Type:        eventType,
			AggregateID: "agg-annule",
			Payload:     []byte(`{}`),
		}); err != nil {
			t.Errorf("Enqueue dans la transaction: %v", err)
			return result.Err[int, string]("enqueue")
		}

		// Le message DOIT être visible ici, sinon un Enqueue qui n'écrirait rien
		// du tout passerait ce test sans qu'on s'en aperçoive.
		var dedans int
		if err := database.QuerierFrom(txCtx, p).QueryRow(txCtx,
			"SELECT count(*) FROM platform.outbox_messages WHERE event_type = $1", eventType,
		).Scan(&dedans); err != nil {
			t.Errorf("comptage dans la transaction: %v", err)
			return result.Err[int, string]("comptage")
		}
		if dedans != 1 {
			t.Errorf("%d message(s) visible(s) DANS la transaction, attendu 1", dedans)
		}

		// L'échec métier qui doit tout annuler.
		return result.Err[int, string]("l'écriture métier a échoué")
	})

	if res.IsOk() {
		t.Fatal("l'unité de travail devait rendre une erreur")
	}

	// Vu depuis le POOL, hors de toute transaction : plus rien.
	var dehors int
	if err := p.QueryRow(ctx,
		"SELECT count(*) FROM platform.outbox_messages WHERE event_type = $1", eventType,
	).Scan(&dehors); err != nil {
		t.Fatalf("comptage hors transaction: %v", err)
	}
	if dehors != 0 {
		t.Fatalf("%d message(s) survivent à l'annulation : un événement partirait pour "+
			"une écriture métier qui n'a jamais eu lieu (ADR 006)", dehors)
	}
}
