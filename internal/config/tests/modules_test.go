package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestModuleAbsentIsDisabled : deny par défaut jusque dans la configuration. On
// n'active jamais une capacité que personne n'a demandée.
func TestModuleAbsentIsDisabled(t *testing.T) {
	t.Parallel()

	got := config.Modules{}.Get("outbox")
	if got.Enabled {
		t.Error("un module absent de la configuration doit être désactivé")
	}
	if got.Driver != "memory" {
		t.Errorf("pilote par défaut = %q, attendu \"memory\" (sans dépendance externe)", got.Driver)
	}
}

// TestMissingDriverFallsBackToZeroDependencyDefault : un module activé sans
// pilote explicite prend le pilote sans dépendance, jamais le plus complet.
func TestMissingDriverFallsBackToZeroDependencyDefault(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"outbox":       "memory",
		"idempotency":  "memory",
		"dynconf":      "file",
		"audit":        "log",
		"storage":      "disk",
		"notification": "log",
		"payment":      "log",
		"secrets":      "env",
	}

	for module, want := range cases {
		t.Run(module, func(t *testing.T) {
			t.Parallel()
			modules := config.Modules{module: {Enabled: true}}
			if got := modules.DriverOf(module); got != want {
				t.Errorf("pilote par défaut de %s = %q, attendu %q", module, got, want)
			}
		})
	}
}

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
				"ratelimit":   {Enabled: true},
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
			if got := tc.modules.RequiresSQL(); got != tc.want {
				t.Errorf("RequiresSQL() = %v, attendu %v", got, tc.want)
			}
		})
	}
}

func TestRequiresCache(t *testing.T) {
	t.Parallel()

	defaults := config.Modules{"idempotency": {Enabled: true}}
	if defaults.RequiresCache() {
		t.Error("les pilotes par défaut ne doivent exiger aucun cache")
	}

	withRedis := config.Modules{"idempotency": {Enabled: true, Driver: "redis"}}
	if !withRedis.RequiresCache() {
		t.Error("un pilote redis actif doit exiger le cache")
	}

	disabled := config.Modules{"idempotency": {Enabled: false, Driver: "redis"}}
	if disabled.RequiresCache() {
		t.Error("un module désactivé n'exige rien")
	}
}

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	modules := config.Modules{"outbox": {Enabled: true}, "audit": {Enabled: false}}
	if !modules.IsEnabled("outbox") {
		t.Error("outbox devrait être actif")
	}
	if modules.IsEnabled("audit") {
		t.Error("audit devrait être inactif")
	}
	if modules.IsEnabled("inexistant") {
		t.Error("un module inconnu ne peut pas être actif")
	}
}
