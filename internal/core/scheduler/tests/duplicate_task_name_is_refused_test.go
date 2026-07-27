package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestDuplicateTaskNameIsRefused : deux tâches de même nom sont refusées.
//
// Le nom EST la clé d'élection. Deux tâches homonymes s'excluraient donc
// mutuellement : l'une des deux ne tournerait jamais, et le seul symptôme serait
// un compte rendu « skipped » — indistinguable du cas nominal d'une réplique non
// élue. Le défaut serait invisible aussi longtemps que personne ne s'aperçoit que
// le travail n'est pas fait.
func TestDuplicateTaskNameIsRefused(t *testing.T) {
	t.Parallel()

	noop := func(context.Context) error { return nil }
	scheduled := []application.Scheduled{
		{Task: task("purge"), Job: noop},
		{Task: task("purge"), Job: noop},
	}

	acquire, release := alwaysElected()
	runner := newRunner(t, acquire, release, &journal{})

	err := runner.Run(context.Background(), scheduled)
	if !errors.Is(err, domain.ErrInvalidTask) {
		t.Errorf("Run = %v, attendu ErrInvalidTask", err)
	}
}
