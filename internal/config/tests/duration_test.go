// Package tests porte les tests EN BOÎTE NOIRE du paquet config.
//
// # Pourquoi un sous-dossier
//
// Ces tests n'accèdent qu'à l'API publique, exactement comme un appelant. Ils
// vérifient donc le contrat, pas l'implémentation — et un refactoring interne ne
// les casse pas.
//
// Corollaire à connaître : un test dans ce dossier ne peut PAS atteindre un
// identifiant non exporté. Les tests de `expand`, `deepMerge` et des `validate()`
// restent dans `internal/config/internal_test.go`, à côté du code.
// Voir rules/tests.md.
package tests

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDurationAcceptsHumanForms verrouille le défaut qui rendait toute la
// configuration inutilisable : yaml.v3 ne décode pas "5s" en time.Duration, il
// n'accepte qu'un entier de nanosecondes.
func TestDurationAcceptsHumanForms(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		yaml string
		want time.Duration
	}{
		"secondes":              {yaml: `d: 5s`, want: 5 * time.Second},
		"heures et minutes":     {yaml: `d: 1h30m`, want: 90 * time.Minute},
		"millisecondes":         {yaml: `d: 250ms`, want: 250 * time.Millisecond},
		"zero explicite":        {yaml: `d: 0s`, want: 0},
		"entier lu en SECONDES": {yaml: `d: 30`, want: 30 * time.Second},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out struct{ D config.Duration }
			if err := yaml.Unmarshal([]byte(tc.yaml), &out); err != nil {
				t.Fatalf("décodage de %q: %v", tc.yaml, err)
			}
			if out.D.Duration() != tc.want {
				t.Errorf("%q → %v, attendu %v", tc.yaml, out.D.Duration(), tc.want)
			}
		})
	}
}

// TestDurationIntegerIsNeverNanoseconds : `read_timeout: 5` doit valoir cinq
// SECONDES. Lu en nanosecondes, le délai vaudrait pratiquement zéro — donc une
// panne silencieuse, le pire des défauts.
func TestDurationIntegerIsNeverNanoseconds(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 5`), &out); err != nil {
		t.Fatalf("décodage: %v", err)
	}
	if out.D.Duration() < time.Second {
		t.Errorf("entier interprété en nanosecondes: %v", out.D.Duration())
	}
}

// TestDurationRefusesGarbage : une durée illisible doit ÉCHOUER, jamais valoir
// zéro. Un délai à zéro n'est pas une valeur sûre.
func TestDurationRefusesGarbage(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`d: cinq secondes`, `d: 5 secondes`, `d: [1,2]`, `d: {a: 1}`} {
		var out struct{ D config.Duration }
		if err := yaml.Unmarshal([]byte(raw), &out); err == nil {
			t.Errorf("%q accepté alors qu'il devrait échouer (obtenu %v)", raw, out.D)
		}
	}
}

// TestDurationRoundTrip : la configuration effective doit pouvoir être réaffichée
// telle qu'elle serait écrite, pour qu'un diff de configuration soit lisible.
func TestDurationRoundTrip(t *testing.T) {
	t.Parallel()

	var out struct{ D config.Duration }
	if err := yaml.Unmarshal([]byte(`d: 1h30m0s`), &out); err != nil {
		t.Fatalf("décodage: %v", err)
	}
	encoded, err := yaml.Marshal(out)
	if err != nil {
		t.Fatalf("encodage: %v", err)
	}

	var back struct{ D config.Duration }
	if err := yaml.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("re-décodage de %q: %v", encoded, err)
	}
	if back.D != out.D {
		t.Errorf("aller-retour non conservatif: %v puis %v", out.D, back.D)
	}
}
