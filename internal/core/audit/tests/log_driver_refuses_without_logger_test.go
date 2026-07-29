package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestLogDriverRefusesWithoutLogger: the default driver requires no external
// SERVICE, but it does require an explicit logger. Falling back on
// slog.Default() would send the audit towards an output nobody has chosen to
// collect.
func TestLogDriverRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	_, err := audit.New(config.Module{Enabled: true, Driver: "log"}, audit.Deps{})
	if !errors.Is(err, audit.ErrLoggerRequired) {
		t.Errorf("error = %v, want ErrLoggerRequired", err)
	}
}
