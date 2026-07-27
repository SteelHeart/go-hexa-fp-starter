package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedDriverOptionsParse : une option de pilote n'est validée qu'à la
// construction du module. Sans ce test, `ttl: 24` — un entier au lieu d'une durée,
// donc 24 secondes au lieu de 24 heures — resterait invisible jusqu'à ce qu'un
// rejeu légitime soit refusé en production.
func TestShippedDriverOptionsParse(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}

	ttl, err := cfg.Modules.Get("idempotency").DurationOption("ttl", 0)
	if err != nil {
		t.Fatalf("options du module idempotency illisibles: %v", err)
	}
	if ttl < time.Hour {
		t.Errorf("ttl livré = %v : une fenêtre de rejeu si courte trahit une unité oubliée", ttl)
	}
}
