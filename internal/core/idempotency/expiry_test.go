package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestExpiredReservationIsTakenOver : une réservation abandonnée — processus tué
// entre Reserve et Complete — doit se reprendre. Sans reprise, un plantage
// bloquerait la clé jusqu'à sa purge et le client ne pourrait plus rien faire.
func TestExpiredReservationIsTakenOver(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Fatalf("avant expiration, attendu ErrInFlight, reçu %v", err)
	}

	clk.advance(time.Hour + time.Second)

	taken, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("après expiration, la clé doit être reprise: %v", err)
	}
	if taken.Replayed {
		t.Error("une réservation expirée n'a rien mémorisé")
	}
}

// TestExpiredMemoryStopsReplaying : la fenêtre de rejeu est finie, et c'est un
// arbitrage assumé. Au-delà du TTL, le rejeu recrée la ressource.
func TestExpiredMemoryStopsReplaying(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "30m")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}

	clk.advance(31 * time.Minute)

	after, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("réservation après expiration: %v", err)
	}
	if after.Replayed {
		t.Error("passé le TTL, la réponse ne doit plus être rejouée")
	}
}

// TestCompleteAfterExpiryReportsNotReserved : l'opération métier a réussi, seule
// la mémorisation échoue. L'erreur doit le dire — l'appelant journalise, il
// n'annule pas.
func TestCompleteAfterExpiryReportsNotReserved(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	clk.advance(2 * time.Hour)

	err := mod.Complete(ctx, req.Key, []byte("ok"))
	if !errors.Is(err, domain.ErrNotReserved) {
		t.Errorf("Complete = %v, attendu ErrNotReserved", err)
	}
}

// TestPurgeRemovesOnlyExpiredKeys : le pilote mémoire n'expire rien tout seul.
// Purge est donc la seule borne à la croissance de la carte, et elle ne doit
// jamais emporter une réservation vivante.
func TestPurgeRemovesOnlyExpiredKeys(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()

	for _, key := range []string{"vieille-1", "vieille-2"} {
		if _, err := mod.Reserve(ctx, request(key, "charge")); err != nil {
			t.Fatalf("réservation de %s: %v", key, err)
		}
	}
	clk.advance(2 * time.Hour)
	if _, err := mod.Reserve(ctx, request("recente", "charge")); err != nil {
		t.Fatalf("réservation récente: %v", err)
	}

	removed, err := mod.Purge(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("clés purgées = %d, attendu 2", removed)
	}
	if _, err := mod.Reserve(ctx, request("recente", "charge")); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("la réservation récente devait survivre à la purge, reçu %v", err)
	}
}

// TestDefaultTTLApplies : sans option, le module retient une fenêtre de rejeu par
// défaut plutôt qu'aucune. Un TTL nul rendrait le module inopérant en silence.
func TestDefaultTTLApplies(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	clk.advance(23 * time.Hour)
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("à 23 h, la réservation par défaut doit tenir, reçu %v", err)
	}
}
