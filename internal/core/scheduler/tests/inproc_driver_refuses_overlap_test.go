package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestInprocDriverRefusesOverlap : le pilote sans dépendance n'élit personne entre
// répliques, mais il refuse le RECOUVREMENT local — et ça, ce n'est pas rien.
//
// Une tâche plus lente que sa période empilerait sinon les exécutions à chaque tick
// jusqu'à saturer le processus. La panne arriverait longtemps après la mise en
// service, sous charge, et ressemblerait à une fuite de mémoire.
func TestInprocDriverRefusesOverlap(t *testing.T) {
	t.Parallel()

	mod := newInprocModule(t)
	ctx := context.Background()
	const name domain.TaskName = "purge"

	first, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("première élection: %v", err)
	}
	if !first {
		t.Fatal("la première élection doit être accordée")
	}

	second, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("seconde élection: %v", err)
	}
	if second {
		t.Error("une tâche déjà en cours ne doit pas être relancée")
	}

	if err := mod.Release(ctx, name); err != nil {
		t.Fatalf("libération: %v", err)
	}
	again, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("élection après libération: %v", err)
	}
	if !again {
		t.Error("après libération, la tâche doit être réélisible")
	}
}
