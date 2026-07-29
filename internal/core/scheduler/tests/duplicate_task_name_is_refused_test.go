package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestDuplicateTaskNameIsRefused: two tasks with the same name are refused.
//
// The name IS the election key. Two tasks with the same name would therefore
// exclude each other: one of the two would never run, and the only symptom
// would be a "skipped" report — indistinguishable from the nominal case of a
// non-elected replica. The defect would stay invisible for as long as nobody
// noticed the work was not being done.
func TestDuplicateTaskNameIsRefused(t *testing.T) {
	t.Parallel()

	noop := func(context.Context) error { return nil }
	scheduled := []application.Scheduled{
		{Task: task("purge"), Job: noop},
		{Task: task("purge"), Job: noop},
	}

	acquire, release := alwaysElected()
	runner := newRunner(t, acquire, release, &reportLog{})

	err := runner.Run(context.Background(), scheduled)
	if !errors.Is(err, domain.ErrInvalidTask) {
		t.Errorf("Run = %v, want ErrInvalidTask", err)
	}
}
