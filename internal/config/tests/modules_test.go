package tests

import (
	"testing"
	"time"

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

// TestDurationOption : les options de pilote ne sont pas typées à la lecture du
// fichier. Cet accesseur est le seul endroit où une durée d'option est
// interprétée, donc le seul endroit à tester — tous les pilotes en héritent.
func TestDurationOption(t *testing.T) {
	t.Parallel()

	const fallback = 24 * time.Hour

	cases := map[string]struct {
		options map[string]any
		want    time.Duration
		refused bool
	}{
		"absente : valeur par défaut":   {options: nil, want: fallback},
		"nulle : valeur par défaut":     {options: map[string]any{"ttl": nil}, want: fallback},
		"chaîne avec unité":             {options: map[string]any{"ttl": "90m"}, want: 90 * time.Minute},
		"entier : des secondes":         {options: map[string]any{"ttl": 30}, want: 30 * time.Second},
		"entier 64 bits : des secondes": {options: map[string]any{"ttl": int64(45)}, want: 45 * time.Second},
		"unité manquante : refus":       {options: map[string]any{"ttl": "24"}, refused: true},
		"négative : refus":              {options: map[string]any{"ttl": "-1h"}, refused: true},
		"nulle en durée : refus":        {options: map[string]any{"ttl": "0s"}, refused: true},
		"type inattendu : refus":        {options: map[string]any{"ttl": true}, refused: true},
		"décimal non supporté : refus":  {options: map[string]any{"ttl": 1.5}, refused: true},
		"autre clé : sans effet":        {options: map[string]any{"namespace": "x"}, want: fallback},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.DurationOption("ttl", fallback)
			if tc.refused {
				if err == nil {
					t.Fatalf("attendu un refus, reçu %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("refus inattendu: %v", err)
			}
			if got != tc.want {
				t.Errorf("DurationOption = %v, attendu %v", got, tc.want)
			}
		})
	}
}

// TestStringOption : une valeur présente mais vide trahit une variable
// d'environnement non substituée, pas une intention. Se rabattre sur la valeur
// par défaut masquerait une configuration cassée.
func TestStringOption(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		options map[string]any
		want    string
		refused bool
	}{
		"absente : valeur par défaut": {options: nil, want: "idempotency"},
		"présente":                    {options: map[string]any{"namespace": "paiements"}, want: "paiements"},
		"vide : refus":                {options: map[string]any{"namespace": ""}, refused: true},
		"type inattendu : refus":      {options: map[string]any{"namespace": 42}, refused: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mod := config.Module{Enabled: true, Options: tc.options}
			got, err := mod.StringOption("namespace", "idempotency")
			if tc.refused {
				if err == nil {
					t.Fatalf("attendu un refus, reçu %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("refus inattendu: %v", err)
			}
			if got != tc.want {
				t.Errorf("StringOption = %q, attendu %q", got, tc.want)
			}
		})
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
