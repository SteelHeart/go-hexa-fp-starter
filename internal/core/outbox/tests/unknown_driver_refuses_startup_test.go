package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
)

func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	// Deny by default: never a fallback on « the closest driver ».
	if _, err := outbox.New(config.Module{Enabled: true, Driver: "postgresql"}, outbox.Deps{}); err == nil {
		t.Error("an unknown driver must refuse startup")
	}
}
