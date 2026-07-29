package tests

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationAcceptsHumanForms locks down the defect that made the whole
// configuration unusable: yaml.v3 does not decode "5s" into a time.Duration, it
// only accepts an integer of nanoseconds.
func TestDurationAcceptsHumanForms(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		yaml string
		want time.Duration
	}{
		"seconds":                 {yaml: `d: 5s`, want: 5 * time.Second},
		"hours and minutes":       {yaml: `d: 1h30m`, want: 90 * time.Minute},
		"milliseconds":            {yaml: `d: 250ms`, want: 250 * time.Millisecond},
		"explicit zero":           {yaml: `d: 0s`, want: 0},
		"integer read in SECONDS": {yaml: `d: 30`, want: 30 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out struct{ D config.Duration }
			if err := yaml.Unmarshal([]byte(tc.yaml), &out); err != nil {
				t.Fatalf("decoding of %q: %v", tc.yaml, err)
			}
			if out.D.Duration() != tc.want {
				t.Errorf("%q → %v, want %v", tc.yaml, out.D.Duration(), tc.want)
			}
		})
	}
}
