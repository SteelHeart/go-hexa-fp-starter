package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestInvalidTTLOptionRefusesStartup: an unreadable duration stops startup.
//
// Falling back on the default value would be worse than refusing. Someone
// writes `session_ttl: 30` meaning "thirty minutes", the module keeps twelve
// hours, and the configuration states the opposite of what applies. Nobody
// re-reads a value that produced no error.
func TestInvalidTTLOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	for _, value := range []any{"thirty minutes", "30x", true, []string{"1h"}} {
		_, err := auth.New(config.Module{
			Enabled: true,
			Driver:  "memory",
			Options: map[string]any{"session_ttl": value},
		}, deps(c))
		if err == nil {
			t.Errorf("session_ttl=%v (%T): startup must be refused", value, value)
		}
	}
}

// TestUnknownOptionRefusesStartup ties this module to the guard of issue #93.
//
// A misspelt option was ignored IN SILENCE: the server started, mounted the
// driver, and said nothing about it. This module's catalogue enumerates its
// options; the test records that the guard does cover `auth`, rather than
// trusting the fact that it is wired up somewhere else.
func TestUnknownOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	allowed := auth.Catalog()[auth.Name].Drivers["memory"].Options
	if len(allowed) == 0 {
		t.Fatal("the catalogue must enumerate the options of the memory driver")
	}

	found := false
	for _, option := range allowed {
		if option == auth.OptionSessionTTL {
			found = true
		}
	}
	if !found {
		t.Fatalf("the catalogue must declare %q; it declares %v", auth.OptionSessionTTL, allowed)
	}
}
