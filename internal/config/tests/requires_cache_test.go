package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

func TestRequiresCache(t *testing.T) {
	t.Parallel()

	defaults := config.Modules{"idempotency": {Enabled: true}}
	if defaults.RequiresCache(shippedCatalog(t)) {
		t.Error("les pilotes par défaut ne doivent exiger aucun cache")
	}

	withRedis := config.Modules{"idempotency": {Enabled: true, Driver: "redis"}}
	if !withRedis.RequiresCache(shippedCatalog(t)) {
		t.Error("un pilote redis actif doit exiger le cache")
	}

	disabled := config.Modules{"idempotency": {Enabled: false, Driver: "redis"}}
	if disabled.RequiresCache(shippedCatalog(t)) {
		t.Error("un module désactivé n'exige rien")
	}
}
