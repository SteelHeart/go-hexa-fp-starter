package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestRequiresSQL verrouille la promesse centrale de l'ADR 012 : avec les
// pilotes par défaut, AUCUNE base n'est nécessaire pour démarrer.
func TestRequiresSQL(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		modules config.Modules
		want    bool
	}{
		"pilotes par defaut, aucune base SQL requise": {
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
		"un seul pilote postgres suffit": {
			modules: config.Modules{"outbox": {Enabled: true, Driver: "postgres"}},
			want:    true,
		},
		"le scheduler en advisory-lock exige une base": {
			modules: config.Modules{"scheduler": {Enabled: true, Driver: "advisory-lock"}},
			want:    true,
		},
		"un pilote postgres sur un module DESACTIVE n'exige rien": {
			modules: config.Modules{"outbox": {Enabled: false, Driver: "postgres"}},
			want:    false,
		},
		"configuration vide": {modules: config.Modules{}, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.modules.RequiresSQL(shippedCatalog(t)); got != tc.want {
				t.Errorf("RequiresSQL() = %v, attendu %v", got, tc.want)
			}
		})
	}
}
