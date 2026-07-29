//go:build integration

package integration

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	pgsched "github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/postgres"
)

// TestSchedulerTwoReplicasNeverRunTheSameTask exercises election between
// replicas.
//
// This is the property the `cron-inproc` driver does NOT have, and which it
// documents as its NON-guarantee: without election, every replica runs the
// task. A nightly purge launched three times over, or a report sent three
// times.
//
// The Postgres advisory lock belongs to its SESSION, hence to its connection.
// That is what makes this driver delicate, and untestable without a database:
// taking it on a connection immediately handed back to the pool amounts to
// locking nothing. Two distinct electors, like two processes, are the only way
// to observe it.
func TestSchedulerTwoReplicasNeverRunTheSameTask(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)

	// Two electors on the same pool: each will take its OWN connection, which
	// faithfully reproduces two replicas.
	replica1 := pgsched.New(p)
	replica2 := pgsched.New(p)

	// A name specific to the test: the lock key derives from it, and a
	// collision with another test would make the failure intermittent.
	task := domain.TaskName(unique(t, "integration-purge"))

	elected1, err := replica1.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("first election: %v", err)
	}
	if !elected1 {
		t.Fatal("the first replica must be elected: nobody holds the lock")
	}

	elected2, err := replica2.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("second election: %v", err)
	}
	if elected2 {
		t.Fatal("BOTH replicas were elected: the task would run twice, " +
			"which is exactly what election has to prevent")
	}

	// After release, the second one must be able to take over — otherwise the
	// task would never run again after the first restart.
	if err := replica1.Release(ctx, task); err != nil {
		t.Fatalf("release: %v", err)
	}

	elected2, err = replica2.Acquire(ctx, task)
	if err != nil {
		t.Fatalf("election after release: %v", err)
	}
	if !elected2 {
		t.Fatal("after release, the second replica must be elected: " +
			"a lock never released freezes the task until restart")
	}
	t.Cleanup(func() { _ = replica2.Release(ctxTest(t), task) })
}
