package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintDistinguishesPayloads : une clé réutilisée avec une charge
// différente doit être détectée. Si les empreintes se confondaient, le second
// appelant recevrait la réponse du premier — une fuite de données.
func TestFingerprintDistinguishesPayloads(t *testing.T) {
	t.Parallel()

	cases := map[string][2]any{
		"valeur différente":    {map[string]int{"montant": 100}, map[string]int{"montant": 101}},
		"champ supplémentaire": {map[string]int{"montant": 100}, map[string]int{"montant": 100, "frais": 0}},
		"type différent":       {map[string]any{"montant": 100}, map[string]any{"montant": "100"}},
		"nil contre vide":      {nil, map[string]any{}},
		"valeurs permutées":    {map[string]int{"a": 1, "b": 2}, map[string]int{"a": 2, "b": 1}},
	}

	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			left, right := domain.Fingerprint(pair[0]), domain.Fingerprint(pair[1])
			if left == right {
				t.Errorf("empreintes identiques pour deux charges différentes: %q", left)
			}
		})
	}
}
