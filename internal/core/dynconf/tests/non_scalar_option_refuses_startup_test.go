package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestNonScalarOptionRefusesStartup: `flags: {a: {b: 1}}` is a typo, not an
// intention. Letting it through would give a silently inactive flag — the
// hardest defect to see on this module, since an inactive flag looks in every
// respect like a flag one has not switched on yet.
func TestNonScalarOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]any{
		"nested table":      {"flags": map[string]any{"a": map[string]any{"b": 1}}},
		"list":              {"settings": map[string]any{"a": []any{1, 2}}},
		"group not a table": {"flags": "new_payment"},
	}

	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := dynconf.New(
				config.Module{Enabled: true, Driver: "file", Options: options},
				dynconf.Deps{},
			)
			if err == nil {
				t.Error("a malformed option must refuse to start")
			}
		})
	}
}
