package outbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// fixedNow rend les tests déterministes : ni horloge réelle, ni attente.
func fixedNow() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

func newMemoryModule(t *testing.T) outbox.Module {
	t.Helper()
	// Deps sans Pool : c'est exactement la promesse de l'ADR 012.
	mod, err := outbox.New(
		config.Module{Enabled: true, Driver: "memory"},
		outbox.Deps{Now: fixedNow},
	)
	if err != nil {
		t.Fatalf("construction du module en pilote memory: %v", err)
	}
	return mod
}

// TestMemoryDriverNeedsNoDatabase verrouille la promesse centrale du socle :
// le pilote par défaut démarre sans aucune connexion.
func TestMemoryDriverNeedsNoDatabase(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: true, Driver: "memory"}, outbox.Deps{})
	if err != nil {
		t.Fatalf("le pilote memory ne doit exiger aucune dépendance: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); err != nil {
		t.Errorf("Enqueue sans base doit réussir: %v", err)
	}
}

func TestPostgresDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	// Refus au DÉMARRAGE, pas à la première requête : un service à moitié
	// configuré échoue plus tard, ailleurs, et pour une raison sans rapport.
	_, err := outbox.New(config.Module{Enabled: true, Driver: "postgres"}, outbox.Deps{})
	if !errors.Is(err, outbox.ErrPoolRequired) {
		t.Errorf("attendu ErrPoolRequired, obtenu: %v", err)
	}
}

func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	// Deny par défaut : jamais de repli sur « le pilote le plus proche ».
	if _, err := outbox.New(config.Module{Enabled: true, Driver: "postgresql"}, outbox.Deps{}); err == nil {
		t.Error("un pilote inconnu doit refuser le démarrage")
	}
}

// TestDisabledModuleFailsLoudly : un module désactivé qui « marcherait quand
// même » est un piège. Un événement silencieusement ignoré ne se signale jamais.
func TestDisabledModuleFailsLoudly(t *testing.T) {
	t.Parallel()

	mod, err := outbox.New(config.Module{Enabled: false}, outbox.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	if _, err := mod.Enqueue(context.Background(), domain.NewMessage{Type: "t"}); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("Enqueue sur module désactivé: attendu ErrDisabled, obtenu %v", err)
	}
	if err := mod.MarkDone(context.Background(), "x"); !errors.Is(err, outbox.ErrDisabled) {
		t.Errorf("MarkDone sur module désactivé: attendu ErrDisabled, obtenu %v", err)
	}
}

func TestEnqueueThenClaimReturnsTheMessage(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()

	id, err := mod.Enqueue(ctx, domain.NewMessage{Type: "user.registered.v1", AggregateID: "u1"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	claimed, err := mod.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != id {
		t.Fatalf("Claim a retourné %d message(s), attendu 1 portant %s", len(claimed), id)
	}
	if claimed[0].Status != domain.StatusPending {
		t.Errorf("statut initial = %q, attendu pending", claimed[0].Status)
	}
}

// TestClaimIsExclusive vérifie le contrat le plus important du port : deux
// réservations concurrentes ne retournent JAMAIS le même message. Le pilote
// postgres l'obtient par FOR UPDATE SKIP LOCKED ; le pilote memory doit le
// garantir aussi, sinon il mentirait sur le comportement de production.
func TestClaimIsExclusive(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	if _, err := mod.Enqueue(ctx, domain.NewMessage{Type: "t"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	first, err := mod.Claim(ctx, 10)
	if err != nil || len(first) != 1 {
		t.Fatalf("première réservation: %d message(s), err=%v", len(first), err)
	}
	second, err := mod.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("seconde réservation: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("un message déjà réservé a été rendu une seconde fois: %d", len(second))
	}
}

func TestMarkDoneRemovesFromPending(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	id, _ := mod.Enqueue(ctx, domain.NewMessage{Type: "t"})

	if count, _ := mod.PendingCount(ctx); count != 1 {
		t.Fatalf("PendingCount = %d, attendu 1", count)
	}
	if err := mod.MarkDone(ctx, id); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}
	if count, _ := mod.PendingCount(ctx); count != 0 {
		t.Errorf("PendingCount après MarkDone = %d, attendu 0", count)
	}
}

// TestMarkFailedReschedulesInTheFuture : après un échec, le message ne doit PAS
// être immédiatement rejouable — sinon le worker boucle à plein régime sur un
// message cassé.
func TestMarkFailedReschedulesInTheFuture(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	if _, err := mod.Enqueue(ctx, domain.NewMessage{Type: "t"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	claimed, _ := mod.Claim(ctx, 1)

	attempt := domain.NextAttempt(claimed[0], 5, time.Second, fixedNow(), "réseau indisponible")
	if err := mod.MarkFailed(ctx, attempt); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	again, _ := mod.Claim(ctx, 10)
	if len(again) != 0 {
		t.Errorf("le message est redevenu réservable immédiatement après un échec")
	}
	if count, _ := mod.PendingCount(ctx); count != 1 {
		t.Errorf("un message en attente de réessai doit rester pending, count=%d", count)
	}
}

func TestNextAttemptBackoffIsExponentialAndBounded(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	base := time.Second

	cases := map[string]struct {
		attempts int
		want     time.Duration
	}{
		"premier echec":  {attempts: 0, want: 2 * time.Second},
		"deuxieme echec": {attempts: 1, want: 4 * time.Second},
		"troisieme":      {attempts: 2, want: 8 * time.Second},
		// Le décalage est borné : sans borne, 1<<40 déborderait et produirait une
		// durée NÉGATIVE, donc un message rejoué en boucle immédiatement.
		"decalage borne a 2^10": {attempts: 40, want: 1024 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := domain.NextAttempt(
				domain.Message{Attempts: tc.attempts}, 100, base, now, "raison",
			)
			if delay := got.AvailableAt.Sub(now); delay != tc.want {
				t.Errorf("recul = %v, attendu %v", delay, tc.want)
			}
			if got.AvailableAt.Before(now) {
				t.Error("le prochain essai est dans le passé : débordement de durée")
			}
		})
	}
}

func TestNextAttemptMarksFailedWhenAttemptsExhausted(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 4}, 5, time.Second, fixedNow(), "raison")
	if got.Status != domain.StatusFailed {
		t.Errorf("statut = %q, attendu failed après épuisement des tentatives", got.Status)
	}

	// Un message abandonné n'est JAMAIS supprimé : c'est la seule trace de ce
	// qui n'a pas été publié.
	if got.Reason == "" {
		t.Error("la raison de l'abandon doit être conservée")
	}
}

func TestNextAttemptStaysPendingBeforeExhaustion(t *testing.T) {
	t.Parallel()

	got := domain.NextAttempt(domain.Message{Attempts: 1}, 5, time.Second, fixedNow(), "raison")
	if got.Status != domain.StatusPending {
		t.Errorf("statut = %q, attendu pending", got.Status)
	}
}

func TestIsDueRespectsStatusAndSchedule(t *testing.T) {
	t.Parallel()

	now := fixedNow()
	cases := map[string]struct {
		msg  domain.Message
		want bool
	}{
		"pending et echu":             {msg: domain.Message{Status: domain.StatusPending, AvailableAt: now}, want: true},
		"pending mais futur":          {msg: domain.Message{Status: domain.StatusPending, AvailableAt: now.Add(time.Minute)}, want: false},
		"deja traite":                 {msg: domain.Message{Status: domain.StatusDone, AvailableAt: now}, want: false},
		"abandonne, jamais rejouable": {msg: domain.Message{Status: domain.StatusFailed, AvailableAt: now}, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.msg.IsDue(now); got != tc.want {
				t.Errorf("IsDue = %v, attendu %v", got, tc.want)
			}
		})
	}
}
