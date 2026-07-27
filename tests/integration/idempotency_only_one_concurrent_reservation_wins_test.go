//go:build integration

package integration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	pgidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
)

// TestIdempotencyOnlyOneConcurrentReservationWins éprouve l'exclusivité contre
// un vrai moteur.
//
// Le pilote `memory` a vingt-quatre tests qui prouvent cette propriété — avec
// un mutex, dans un seul processus. Le pilote `postgres` la confie à
// `ON CONFLICT … DO UPDATE … WHERE expires_at < now()`, une construction dont
// la justesse ne se déduit pas : elle se constate.
//
// Ce que ça garde : un client mobile qui réémet sa requête sur un réseau
// instable, et deux répliques qui la reçoivent en même temps. Si les deux
// obtiennent la réservation, l'opération s'exécute deux fois. Sur un paiement,
// c'est un débit en double.
//
// Seize appels concurrents sur la MÊME clé ; exactement un doit gagner.
func TestIdempotencyOnlyOneConcurrentReservationWins(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgidem.New(p, time.Hour)

	key := domain.Key(unique(t, "integration-idem"))
	req := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]string{"a": "b"})}

	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.idempotency_keys WHERE key = $1", key.String())
	})

	const appels = 16
	var (
		attente  sync.WaitGroup
		mu       sync.Mutex
		gagnants int
		enCours  int
		autres   []error
		depart   = make(chan struct{})
	)

	for range appels {
		attente.Add(1)
		go func() {
			defer attente.Done()
			// Tous partent au même instant : sans cette barrière, les appels
			// s'enchaîneraient et la concurrence ne serait jamais éprouvée.
			<-depart

			res, err := store.Reserve(ctx, req)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && !res.Replayed:
				gagnants++
			case errors.Is(err, domain.ErrInFlight):
				enCours++
			default:
				autres = append(autres, err)
			}
		}()
	}
	close(depart)
	attente.Wait()

	if len(autres) > 0 {
		t.Fatalf("%d appel(s) ont échoué autrement qu'en « déjà en cours » : %v", len(autres), autres)
	}
	if gagnants != 1 {
		t.Fatalf("%d réservations obtenues sur %d appels concurrents, attendu exactement 1 — "+
			"l'opération s'exécuterait %d fois", gagnants, appels, gagnants)
	}
	if enCours != appels-1 {
		t.Errorf("%d appel(s) refusés en « déjà en cours », attendu %d", enCours, appels-1)
	}
}
