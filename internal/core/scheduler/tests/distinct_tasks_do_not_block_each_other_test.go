package tests

import (
	"context"
	"testing"
)

// TestDistinctTasksDoNotBlockEachOther : deux tâches différentes s'exécutent en
// parallèle. L'exclusion porte sur UNE tâche, jamais sur l'ordonnanceur entier.
//
// Un verrou global serait plus simple à écrire et catastrophique à exploiter : une
// tâche longue empêcherait toutes les autres de tourner, et le symptôme —
// « certaines tâches ne partent plus » — ne désignerait pas la coupable.
func TestDistinctTasksDoNotBlockEachOther(t *testing.T) {
	t.Parallel()

	mod := newInprocModule(t)
	ctx := context.Background()

	elected, err := mod.Acquire(ctx, "purge")
	if err != nil || !elected {
		t.Fatalf("élection de purge: elected=%v err=%v", elected, err)
	}

	elected, err = mod.Acquire(ctx, "rappels")
	if err != nil {
		t.Fatalf("élection de rappels: %v", err)
	}
	if !elected {
		t.Error("une autre tâche ne doit pas être bloquée par la première")
	}
}
