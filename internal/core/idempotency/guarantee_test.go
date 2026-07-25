package idempotency_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestEmptyKeyIsRefused est le test le plus important du module.
//
// Une clé vide serait partagée par TOUS les appelants : le premier rejeu de
// n'importe qui retournerait la réponse mémorisée de n'importe quel autre. Une
// protection contre les doublons se transformerait en fuite de données entre
// utilisateurs.
func TestEmptyKeyIsRefused(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	cases := map[string]domain.Request{
		"clé vide":               {Key: "", Fingerprint: "abc"},
		"empreinte vide":         {Key: "k1", Fingerprint: ""},
		"les deux vides":         {},
		"clé vide, empreinte ok": {Key: "", Fingerprint: domain.Fingerprint("charge")},
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrIncomplete) {
				t.Errorf("Reserve = %v, attendu ErrIncomplete", err)
			}
		})
	}
}

// TestSecondReservationIsRefusedWhileInFlight : c'est LA garantie du module. Sans
// ce refus, deux requêtes concurrentes exécuteraient toutes les deux l'opération.
func TestSecondReservationIsRefusedWhileInFlight(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("paiement-1", map[string]int{"montant": 4200})

	first, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("première réservation refusée: %v", err)
	}
	if first.Replayed {
		t.Fatal("la première réservation ne peut pas être un rejeu")
	}

	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("seconde réservation = %v, attendu ErrInFlight", err)
	}
}

// TestSameKeyWithDifferentPayloadConflicts : la même clé doit désigner la même
// requête. On refuse plutôt que de deviner laquelle des deux est la bonne.
func TestSameKeyWithDifferentPayloadConflicts(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", map[string]int{"montant": 100})); err != nil {
		t.Fatalf("première réservation refusée: %v", err)
	}
	_, err := mod.Reserve(ctx, request("k1", map[string]int{"montant": 999}))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("erreur = %v, attendu ErrConflict", err)
	}
}

// TestCompletedRequestIsReplayed : après mémorisation, le rejeu rend la réponse
// du premier appel sans rien réexécuter.
func TestCompletedRequestIsReplayed(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte(`{"id":"user-42"}`)); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("le rejeu ne doit pas échouer: %v", err)
	}
	if !replay.Replayed {
		t.Fatal("Replayed doit être vrai : l'appelant ne doit PAS réexécuter")
	}
	if string(replay.Response) != `{"id":"user-42"}` {
		t.Errorf("réponse mémorisée = %q", replay.Response)
	}
}

// TestReleaseAllowsRetryAfterFailure : sans libération, une erreur transitoire
// rendrait l'opération impossible jusqu'à expiration de la clé — le remède serait
// pire que le mal.
func TestReleaseAllowsRetryAfterFailure(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("libération: %v", err)
	}

	again, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("après libération, la clé doit être réservable: %v", err)
	}
	if again.Replayed {
		t.Error("une clé libérée n'a rien mémorisé : Replayed doit être faux")
	}
}

// TestReleaseNeverFreesACompletedKey : libérer une clé achevée rouvrirait la porte
// au rejeu que le module existe pour fermer.
func TestReleaseNeverFreesACompletedKey(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("libération: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("réservation après libération: %v", err)
	}
	if !replay.Replayed {
		t.Error("la mémorisation doit survivre à une libération")
	}
}

// TestReserveIsExclusiveUnderConcurrency : le contrat de ports.Reserve exige que
// deux appels concurrents sur une clé libre ne l'obtiennent JAMAIS tous les deux.
// C'est la seule promesse du module, donc le seul test qui la prouve.
func TestReserveIsExclusiveUnderConcurrency(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	req := request("k1", "charge")

	const racers = 16
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		granted int
		refused int
	)
	wg.Add(racers)
	for range racers {
		go func() {
			defer wg.Done()
			_, err := mod.Reserve(context.Background(), req)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				granted++
			case errors.Is(err, domain.ErrInFlight):
				refused++
			}
		}()
	}
	wg.Wait()

	if granted != 1 {
		t.Errorf("réservations accordées = %d, attendu exactement 1", granted)
	}
	if refused != racers-1 {
		t.Errorf("refus ErrInFlight = %d, attendu %d", refused, racers-1)
	}
}

// TestMemorizedResponseIsCopied : sans recopie, l'appelant garderait une référence
// sur la mémoire du magasin et pourrait altérer une réponse mémorisée. Un rejeu
// rendrait alors autre chose que le premier appel, silencieusement.
func TestMemorizedResponseIsCopied(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	response := []byte("original")
	if err := mod.Complete(ctx, req.Key, response); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}
	response[0] = 'X' // l'appelant réutilise son tampon

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("rejeu: %v", err)
	}
	if string(replay.Response) != "original" {
		t.Errorf("réponse mémorisée = %q, altérée par l'appelant", replay.Response)
	}

	replay.Response[0] = 'Y' // et dans l'autre sens
	second, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("second rejeu: %v", err)
	}
	if string(second.Response) != "original" {
		t.Errorf("réponse mémorisée = %q, altérée par un rejeu précédent", second.Response)
	}
}
