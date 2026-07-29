package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedDispatcherPolicyIsSane: the SHIPPED dispatching policy must be
// operable.
//
// These values are only validated at the construction of the dispatcher, that
// is to say at the startup of a worker that does not exist yet. Without this
// test, `base_backoff: 1` instead of `1s` — hence a second turned into a
// nanosecond — would pass `task check` and would only show up in production, in
// the shape of a tight loop of retries against a broker that is already down.
func TestShippedDispatcherPolicyIsSane(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	outbox := cfg.Modules.Get("outbox")

	batchSize, err := outbox.IntOption("batch_size", 0)
	if err != nil {
		t.Fatalf("batch_size unreadable: %v", err)
	}
	if batchSize < 1 || batchSize > 10_000 {
		t.Errorf("shipped batch_size = %d: out of any reasonable range", batchSize)
	}

	maxAttempts, err := outbox.IntOption("max_attempts", 0)
	if err != nil {
		t.Fatalf("max_attempts unreadable: %v", err)
	}
	if maxAttempts < 2 {
		t.Errorf("shipped max_attempts = %d: a single attempt cancels the whole outbox pattern", maxAttempts)
	}

	backoff, err := outbox.DurationOption("base_backoff", 0)
	if err != nil {
		t.Fatalf("base_backoff unreadable: %v", err)
	}
	if backoff < 100*time.Millisecond {
		t.Errorf("shipped base_backoff = %v: a unit has probably been forgotten", backoff)
	}

	interval, err := outbox.DurationOption("interval", 0)
	if err != nil {
		t.Fatalf("interval unreadable: %v", err)
	}
	if interval < 100*time.Millisecond {
		t.Errorf("shipped interval = %v: the dispatcher would hammer the store", interval)
	}
}
