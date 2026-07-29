package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationOption: driver options are not typed when the file is read. This
// accessor is the only place where an option duration is interpreted, hence the
// only place to test — every driver inherits from it.
func TestDurationOption(t *testing.T) {
	t.Parallel()

	const fallback = 24 * time.Hour

	cases := map[string]struct {
		options map[string]any
		want    time.Duration
		refused bool
	}{
		"absent: default value":          {options: nil, want: fallback},
		"null: default value":            {options: map[string]any{"ttl": nil}, want: fallback},
		"string with a unit":             {options: map[string]any{"ttl": "90m"}, want: 90 * time.Minute},
		"integer: seconds":               {options: map[string]any{"ttl": 30}, want: 30 * time.Second},
		"64-bit integer: seconds":        {options: map[string]any{"ttl": int64(45)}, want: 45 * time.Second},
		"missing unit: refusal":          {options: map[string]any{"ttl": "24"}, refused: true},
		"negative: refusal":              {options: map[string]any{"ttl": "-1h"}, refused: true},
		"zero as a duration: refusal":    {options: map[string]any{"ttl": "0s"}, refused: true},
		"unexpected type: refusal":       {options: map[string]any{"ttl": true}, refused: true},
		"decimal not supported: refusal": {options: map[string]any{"ttl": 1.5}, refused: true},
		"another key: no effect":         {options: map[string]any{"namespace": "x"}, want: fallback},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.DurationOption("ttl", fallback)
			if tc.refused {
				if err == nil {
					t.Fatalf("want a refusal, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if got != tc.want {
				t.Errorf("DurationOption = %v, want %v", got, tc.want)
			}
		})
	}
}
