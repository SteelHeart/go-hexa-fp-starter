package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestRequiresSQL locks down the central promise of ADR 012: with the default
// drivers, NO database is needed in order to start.
func TestRequiresSQL(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		modules config.Modules
		want    bool
	}{
		"default drivers, no SQL database required": {
			modules: config.Modules{
				"outbox":      {Enabled: true},
				"idempotency": {Enabled: true},
				"audit":       {Enabled: true},
				"dynconf":     {Enabled: true},
				"storage":     {Enabled: true},
				"scheduler":   {Enabled: true},
			},
			want: false,
		},
		"a single postgres driver is enough": {
			modules: config.Modules{"outbox": {Enabled: true, Driver: "postgres"}},
			want:    true,
		},
		"the scheduler on advisory-lock requires a database": {
			modules: config.Modules{"scheduler": {Enabled: true, Driver: "advisory-lock"}},
			want:    true,
		},
		"a postgres driver on a DISABLED module requires nothing": {
			modules: config.Modules{"outbox": {Enabled: false, Driver: "postgres"}},
			want:    false,
		},
		"empty configuration": {modules: config.Modules{}, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.modules.RequiresSQL(shippedCatalog(t)); got != tc.want {
				t.Errorf("RequiresSQL() = %v, want %v", got, tc.want)
			}
		})
	}
}
