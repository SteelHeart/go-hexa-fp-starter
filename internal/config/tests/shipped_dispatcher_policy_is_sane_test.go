package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedDispatcherPolicyIsSane : la politique de dépilage LIVRÉE doit être
// exploitable.
//
// Ces valeurs ne sont validées qu'à la construction du dépileur, c'est-à-dire au
// démarrage d'un worker qui n'existe pas encore. Sans ce test, `base_backoff: 1`
// au lieu de `1s` — donc une seconde devenue une nanoseconde — passerait `task
// check` et ne se manifesterait qu'en production, sous forme d'une boucle serrée
// de réessais sur un courtier déjà en panne.
func TestShippedDispatcherPolicyIsSane(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	outbox := cfg.Modules.Get("outbox")

	batchSize, err := outbox.IntOption("batch_size", 0)
	if err != nil {
		t.Fatalf("batch_size illisible: %v", err)
	}
	if batchSize < 1 || batchSize > 10_000 {
		t.Errorf("batch_size livré = %d : hors de toute plage raisonnable", batchSize)
	}

	maxAttempts, err := outbox.IntOption("max_attempts", 0)
	if err != nil {
		t.Fatalf("max_attempts illisible: %v", err)
	}
	if maxAttempts < 2 {
		t.Errorf("max_attempts livré = %d : une seule tentative annule tout le patron outbox", maxAttempts)
	}

	backoff, err := outbox.DurationOption("base_backoff", 0)
	if err != nil {
		t.Fatalf("base_backoff illisible: %v", err)
	}
	if backoff < 100*time.Millisecond {
		t.Errorf("base_backoff livré = %v : une unité a probablement été oubliée", backoff)
	}

	interval, err := outbox.DurationOption("interval", 0)
	if err != nil {
		t.Fatalf("interval illisible: %v", err)
	}
	if interval < 100*time.Millisecond {
		t.Errorf("interval livré = %v : le dépileur martèlerait le magasin", interval)
	}
}
