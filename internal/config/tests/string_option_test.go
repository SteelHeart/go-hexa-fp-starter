package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestStringOption: a value that is present but empty betrays an unsubstituted
// environment variable, not an intention. Falling back on the default value
// would hide a broken configuration.
func TestStringOption(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		options map[string]any
		want    string
		refused bool
	}{
		"absent: default value":    {options: nil, want: "idempotency"},
		"present":                  {options: map[string]any{"namespace": "payments"}, want: "payments"},
		"empty: refusal":           {options: map[string]any{"namespace": ""}, refused: true},
		"unexpected type: refusal": {options: map[string]any{"namespace": 42}, refused: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.StringOption("namespace", "idempotency")
			if tc.refused {
				if err == nil {
					t.Fatalf("want a refusal, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if got != tc.want {
				t.Errorf("StringOption = %q, want %q", got, tc.want)
			}
		})
	}
}
