package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

func TestRequiresCache(t *testing.T) {
	t.Parallel()

	defaults := config.Modules{"idempotency": {Enabled: true}}
	if defaults.RequiresCache(shippedCatalog(t)) {
		t.Error("the default drivers must require no cache")
	}

	withRedis := config.Modules{"idempotency": {Enabled: true, Driver: "redis"}}
	if !withRedis.RequiresCache(shippedCatalog(t)) {
		t.Error("an active redis driver must require the cache")
	}

	disabled := config.Modules{"idempotency": {Enabled: false, Driver: "redis"}}
	if disabled.RequiresCache(shippedCatalog(t)) {
		t.Error("a disabled module requires nothing")
	}
}
