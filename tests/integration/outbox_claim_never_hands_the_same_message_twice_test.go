//go:build integration

package integration

import (
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	pgoutbox "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
)

// TestOutboxClaimNeverHandsTheSameMessageTwice éprouve le cœur du mécanisme :
// `FOR UPDATE SKIP LOCKED`.
//
// C'est LA propriété qu'aucun test unitaire ne peut atteindre. Le pilote
// `memory` la simule ; ici, ce sont deux transactions Postgres réelles qui se
// disputent les mêmes lignes.
//
// Ce qui casserait sans elle : deux répliques du dépileur publieraient le même
// événement. Le consommateur verrait un doublon — et un doublon d'événement
// métier, c'est un courriel envoyé deux fois, ou pire, un paiement rejoué.
//
// Le test réserve depuis DEUX transactions concurrentes, gardées ouvertes
// pendant la réservation de l'autre. Sans `SKIP LOCKED`, la seconde attendrait
// le verrou puis rendrait les mêmes lignes ; avec, elle les saute.
func TestOutboxClaimNeverHandsTheSameMessageTwice(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgoutbox.New(p)

	// Un type d'événement propre à ce test : la table est partagée, et un autre
	// test qui tourne en parallèle ne doit pas fausser le compte.
	eventType := unique(t, "integration.claim")

	const total = 6
	for range total {
		if _, err := store.Enqueue(ctx, domain.NewMessage{
			Type:        eventType,
			AggregateID: "agg-1",
			Payload:     []byte(`{"k":"v"}`),
		}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), //nolint:errcheck // nettoyage, l'échec ne dit rien d'utile
			"DELETE FROM platform.outbox_messages WHERE event_type = $1", eventType)
	})

	// Deux transactions, ouvertes en même temps. Chacune réserve, puis attend
	// que l'autre ait réservé avant de valider : c'est cette simultanéité qui
	// met `SKIP LOCKED` à l'épreuve.
	var attente sync.WaitGroup
	vus := make([][]domain.MessageID, 2)
	var mu sync.Mutex
	pret := make(chan struct{})

	for i := range 2 {
		attente.Add(1)
		go func() {
			defer attente.Done()

			tx, err := p.Begin(ctx)
			if err != nil {
				t.Errorf("Begin: %v", err)
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()

			rows, err := tx.Query(ctx, `
				SELECT id FROM platform.outbox_messages
				WHERE status = 'pending' AND event_type = $1
				ORDER BY available_at LIMIT 3
				FOR UPDATE SKIP LOCKED`, eventType)
			if err != nil {
				t.Errorf("Query: %v", err)
				return
			}
			var ids []domain.MessageID
			for rows.Next() {
				var id domain.MessageID
				if err := rows.Scan(&id); err != nil {
					t.Errorf("Scan: %v", err)
					rows.Close()
					return
				}
				ids = append(ids, id)
			}
			rows.Close()

			mu.Lock()
			vus[i] = ids
			mu.Unlock()

			// Les deux transactions restent ouvertes jusqu'ici : sans cette
			// barrière, la première pourrait relâcher ses verrous avant que la
			// seconde ne réserve, et le test passerait sans rien prouver.
			<-pret
		}()
	}

	// Laisse les deux réserver, puis les libère.
	for {
		mu.Lock()
		fait := len(vus[0]) > 0 && len(vus[1]) > 0
		mu.Unlock()
		if fait {
			break
		}
	}
	close(pret)
	attente.Wait()

	if len(vus[0]) == 0 || len(vus[1]) == 0 {
		t.Fatalf("une transaction n'a rien réservé : %d et %d — le test ne prouverait rien",
			len(vus[0]), len(vus[1]))
	}

	croises := map[domain.MessageID]bool{}
	for _, id := range vus[0] {
		croises[id] = true
	}
	for _, id := range vus[1] {
		if croises[id] {
			t.Fatalf("le message %s a été réservé par les DEUX transactions : "+
				"FOR UPDATE SKIP LOCKED ne protège plus, un dépileur répliqué publierait des doublons", id)
		}
	}
	if got := len(vus[0]) + len(vus[1]); got != total {
		t.Errorf("%d messages réservés au total, attendu %d — des lignes ont été perdues ou dupliquées",
			got, total)
	}
}
