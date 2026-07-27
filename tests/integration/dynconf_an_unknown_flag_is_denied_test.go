//go:build integration

package integration

import (
	"log/slog"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
	pgdyn "github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/postgres"
)

// TestDynconfAnUnknownFlagIsDenied éprouve le deny-par-défaut jusque dans la
// base.
//
// C'est la propriété la plus importante de ce module et la plus facile à
// perdre : un drapeau absent, une base injoignable, une requête en erreur —
// tous doivent rendre ÉTEINT. Rendre « allumé » sur une panne activerait une
// fonctionnalité que personne n'a demandée, au pire moment.
//
// Le test vérifie les deux sens contre la vraie table : absent = éteint,
// écrit = allumé après invalidation du cache.
//
// Le TTL est mis à zéro : ce module met les valeurs en cache, et sans ça le
// test mesurerait la fraîcheur du cache plutôt que le contenu de la base.
func TestDynconfAnUnknownFlagIsDenied(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	logger := slog.New(slog.DiscardHandler)
	store := pgdyn.New(p, logger, 0, time.Now)

	absent := domain.FlagKey(unique(t, "integration-absent"))
	if store.Flag(ctx, absent) {
		t.Fatal("un drapeau ABSENT doit valoir éteint : deny par défaut")
	}

	present := domain.FlagKey(unique(t, "integration-present"))
	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t),
			"DELETE FROM platform.dynamic_config WHERE key = $1", string(present))
	})

	if err := store.Set(ctx, domain.Change{
		Kind:  domain.KindFlag,
		Key:   string(present),
		Value: "true",
	}); err != nil {
		t.Fatalf("écriture du drapeau: %v", err)
	}
	store.Invalidate()

	if !store.Flag(ctx, present) {
		t.Fatal("un drapeau écrit à `true` doit valoir allumé : sinon aucune " +
			"bascule à chaud ne fonctionne, et le module ne sert à rien")
	}

	// Réécriture à `false` : la bascule doit se faire dans les deux sens. Un
	// module qui ne sait qu'allumer ne permet pas d'éteindre en incident.
	if err := store.Set(ctx, domain.Change{
		Kind:  domain.KindFlag,
		Key:   string(present),
		Value: "false",
	}); err != nil {
		t.Fatalf("réécriture du drapeau: %v", err)
	}
	store.Invalidate()

	if store.Flag(ctx, present) {
		t.Fatal("un drapeau rebasculé à `false` doit valoir éteint — c'est le sens " +
			"qu'on emprunte pendant un incident")
	}
}
