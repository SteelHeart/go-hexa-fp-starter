//go:build integration

package integration

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	pgsched "github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/postgres"
)

// TestSchedulerTwoReplicasNeverRunTheSameTask éprouve l'élection entre
// répliques.
//
// C'est la propriété que le pilote `cron-inproc` n'a PAS, et qu'il documente
// comme sa NON-garantie : sans élection, chaque réplique exécute la tâche. Une
// purge nocturne lancée en triple, ou un rapport envoyé trois fois.
//
// Le verrou consultatif de Postgres appartient à sa SESSION, donc à sa
// connexion. C'est ce qui rend ce pilote délicat, et intestable sans base : le
// prendre sur une connexion aussitôt rendue au pool revient à ne rien
// verrouiller. Deux électeurs distincts, comme deux processus, sont le seul
// moyen de le constater.
func TestSchedulerTwoReplicasNeverRunTheSameTask(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	// Deux électeurs sur le même pool : chacun prendra sa PROPRE connexion, ce
	// qui reproduit fidèlement deux répliques.
	replique1 := pgsched.New(p)
	replique2 := pgsched.New(p)

	// Un nom propre au test : la clé de verrou en dérive, et une collision avec
	// un autre test rendrait l'échec intermittent.
	task := domain.TaskName(unique(t, "integration-purge"))

	elu1, err := replique1.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("première élection: %v", err)
	}
	if !elu1 {
		t.Fatal("la première réplique doit être élue : personne ne tient le verrou")
	}

	elu2, err := replique2.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("seconde élection: %v", err)
	}
	if elu2 {
		t.Fatal("les DEUX répliques ont été élues : la tâche s'exécuterait en double, " +
			"ce qui est exactement ce que l'élection doit empêcher")
	}

	// Après libération, la seconde doit pouvoir prendre la main — sinon la tâche
	// ne tournerait plus jamais après le premier redémarrage.
	if err := replique1.Release(ctx, task); err != nil {
		t.Fatalf("libération: %v", err)
	}

	elu2, err = replique2.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("élection après libération: %v", err)
	}
	if !elu2 {
		t.Fatal("après libération, la seconde réplique doit être élue : " +
			"un verrou jamais relâché fige la tâche jusqu'au redémarrage")
	}
	t.Cleanup(func() { _ = replique2.Release(ctxTest(t), task) })
}
