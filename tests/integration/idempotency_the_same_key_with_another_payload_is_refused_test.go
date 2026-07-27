//go:build integration

package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	pgidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
)

// TestIdempotencyTheSameKeyWithAnotherPayloadIsRefused : une clé ne couvre
// qu'UNE requête.
//
// Le mode de défaillance évité est discret et grave. Un client réutilise sa clé
// d'idempotence — par recopie, par génération fautive, ou par malveillance —
// avec une charge utile différente. Sans cette vérification, le socle rendrait
// la réponse mémorisée de la PREMIÈRE requête pour la SECONDE. Le client
// croirait son second virement effectué ; il ne l'aurait jamais été.
//
// Le refus doit donc être explicite, et distinct de « déjà en cours ».
func TestIdempotencyTheSameKeyWithAnotherPayloadIsRefused(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgidem.New(p, time.Hour)

	key := domain.Key(unique(t, "integration-conflit"))
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.idempotency_keys WHERE key = $1", key.String())
	})

	premiere := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]int{"montant": 100})}
	if _, err := store.Reserve(ctx, premiere); err != nil {
		t.Fatalf("première réservation: %v", err)
	}
	if err := store.Complete(ctx, key, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Rejeu à l'identique : la réponse mémorisée revient, et l'opération ne doit
	// PAS être réexécutée.
	rejeu, err := store.Reserve(ctx, premiere)
	if err != nil {
		t.Fatalf("rejeu à l'identique: %v", err)
	}
	if !rejeu.Replayed {
		t.Fatal("un rejeu à l'identique doit être signalé comme rejeu, sinon l'opération se réexécute")
	}
	if string(rejeu.Response) != `{"ok":true}` {
		t.Errorf("réponse mémorisée = %q", rejeu.Response)
	}

	// Même clé, AUTRE charge utile : refus.
	seconde := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]int{"montant": 999})}
	if _, err := store.Reserve(ctx, seconde); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("erreur = %v, attendu ErrConflict — la réponse d'une AUTRE requête "+
			"serait rendue au client, qui croirait son opération effectuée", err)
	}
}
