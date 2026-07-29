package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestIntOption: zero and negative values are refused.
//
// No integer option of the starter makes sense at zero: a batch of zero
// messages, or zero allowed attempts, describe a component that runs without
// doing anything. Refusal at startup is better than a dispatcher polling
// endlessly a store out of which nothing will ever come.
func TestIntOption(t *testing.T) {
	t.Parallel()

	const fallback = 50

	cases := map[string]struct {
		options map[string]any
		want    int
		refused bool
	}{
		"absent: default value": {options: nil, want: fallback},
		"null: default value":   {options: map[string]any{"batch_size": nil}, want: fallback},
		"integer":               {options: map[string]any{"batch_size": 20}, want: 20},
		"64-bit integer":        {options: map[string]any{"batch_size": int64(30)}, want: 30},
		"zero: refusal":         {options: map[string]any{"batch_size": 0}, refused: true},
		"negative: refusal":     {options: map[string]any{"batch_size": -1}, refused: true},
		"string: refusal":       {options: map[string]any{"batch_size": "50"}, refused: true},
		"decimal: refusal":      {options: map[string]any{"batch_size": 1.5}, refused: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.IntOption("batch_size", fallback)
			if tc.refused {
				if err == nil {
					t.Fatalf("want a refusal, got %d", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if got != tc.want {
				t.Errorf("IntOption = %d, want %d", got, tc.want)
			}
		})
	}
}
