//go:build integration

package integration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	redisidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/redis"
)

// TestIdempotencyRedisOnlyOneConcurrentReservationWins éprouve la même
// exclusivité que le pilote Postgres, contre un moteur radicalement différent.
//
// Les deux pilotes portent le même contrat et l'obtiennent par des moyens sans
// rapport : `ON CONFLICT … WHERE expires_at < now()` d'un côté, `SET NX` de
// l'autre. Vérifier l'un ne dit rien de l'autre — c'est précisément ce qu'un
// port en type fonction rend possible, et ce qu'il rend obligatoire de tester
// deux fois.
//
// Ce test est le seul du dépôt qui touche Redis : le paquet `cache` et ce
// pilote n'avaient aucun test, à aucun niveau (#37).
func TestIdempotencyRedisOnlyOneConcurrentReservationWins(t *testing.T) {
	ctx := ctxTest(t)
	client := redisClient(t)

	namespace := unique(t, "integration-idem-redis")
	store := redisidem.New(client, namespace, time.Hour)

	key := domain.Key("cle-partagee")
	req := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]string{"a": "b"})}

	t.Cleanup(func() {
		client.Del(ctxTest(t), namespace+":"+key.String())
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
