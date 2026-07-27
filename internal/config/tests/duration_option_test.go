package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

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
