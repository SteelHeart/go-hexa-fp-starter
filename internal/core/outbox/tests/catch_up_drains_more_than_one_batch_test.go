package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// TestCatchUpDrainsMoreThanOneBatch : un lot PLEIN signifie « il en reste ».
//
// Attendre le tick suivant ferait avancer le rattrapage d'un seul lot par période.
// Si le débit d'entrée dépasse un lot par tour — ce qui arrive dès qu'un
// consommateur revient après une panne — le retard ne se résorberait jamais et
// grandirait au contraire.
//
// Le rattrapage est borné : sans borne, cent mille messages en retard
// monopoliseraient la boucle, le `select` ne serait plus atteint, et le worker
// ignorerait la demande d'arrêt jusqu'à avoir tout vidé.
func TestCatchUpDrainsMoreThanOneBatch(t *testing.T) {
	t.Parallel()

	const batchSize = 2

	var mu sync.Mutex
	rounds := 0
	// Deux lots pleins, puis un lot partiel qui signale la fin du retard.
	claim := func(context.Context, int) ([]domain.Message, error) {
		mu.Lock()
		defer mu.Unlock()
		rounds++
		switch rounds {
		case 1:
			return []domain.Message{pending("m-1", 0), pending("m-2", 0)}, nil
		case 2:
			return []domain.Message{pending("m-3", 0), pending("m-4", 0)}, nil
		default:
			return []domain.Message{pending("m-5", 0)}, nil
		}
	}

	observed := &spy{}
	var handle ports.Handler = func(context.Context, domain.Message) error { return nil }

	policy := testPolicy()
	policy.BatchSize = batchSize
	dispatcher := newDispatcher(t, dispatcherPorts(observed, claim, handle), policy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Trois appels successifs reproduisent ce que fait un tour de scrutation :
	// enchaîner tant que le lot revient plein.
	for range 3 {
		count, err := dispatcher.DrainOnce(ctx)
		if err != nil {
			t.Fatalf("DrainOnce: %v", err)
		}
		if count == 0 {
			break
		}
	}

	if len(observed.done) != 5 {
		t.Errorf("messages traités = %d, attendu 5 — le retard n'a pas été rattrapé", len(observed.done))
	}
}
