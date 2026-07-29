package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestInvalidTaskRefusesBeforeAnyStart: everything is validated BEFORE anything
// at all starts.
//
// Launching three tasks out of four then failing would leave a half-alive
// scheduler: some tasks are running, the caller has received an error, and
// nobody knows what state things are in. That is the hardest state to diagnose,
// hence the one we refuse to produce.
//
// A null period deserves a mention of its own: `time.NewTicker(0)` PANICS, and
// a panic in a scheduler goroutine takes the whole process with it.
func TestInvalidTaskRefusesBeforeAnyStart(t *testing.T) {
	t.Parallel()

	valid := application.Scheduled{Task: task("good"), Job: func(context.Context) error { return nil }}
	noop := func(context.Context) error { return nil }

	cases := map[string][]application.Scheduled{
		"null period": {{
			Task: domain.Task{Name: "no-period", Every: 0}, Job: noop,
		}},
		"negative period": {{
			Task: domain.Task{Name: "negative", Every: -time.Second}, Job: noop,
		}},
		"name missing": {{
			Task: domain.Task{Every: time.Hour}, Job: noop,
		}},
		"negative timeout": {{
			Task: domain.Task{Name: "timeout", Every: time.Hour, Timeout: -time.Second}, Job: noop,
		}},
		"the bad one is the second": {valid, {
			Task: domain.Task{Name: "no-period", Every: 0}, Job: noop,
		}},
	}

	for name, scheduled := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			acquire, release := alwaysElected()
			runner := newRunner(t, acquire, release, &reportLog{})

			err := runner.Run(context.Background(), scheduled)
			if !errors.Is(err, domain.ErrInvalidTask) {
				t.Errorf("Run = %v, want ErrInvalidTask", err)
			}
		})
	}
}
