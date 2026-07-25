package idempotency_test

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintIsDeterministic : sans déterminisme, la même requête rejouée
// produirait une empreinte différente et serait vue comme un conflit. Le module
// refuserait alors des rejeux légitimes — exactement l'inverse de son rôle.
func TestFingerprintIsDeterministic(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"email": "a@example.com", "montant": 4200, "actif": true}
	first := domain.Fingerprint(payload)

	for range 20 {
		if got := domain.Fingerprint(payload); got != first {
			t.Fatalf("empreinte instable: %q puis %q", first, got)
		}
	}
}

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

// TestFingerprintIgnoresMapOrder : deux constructions de la même map doivent
// rendre la même empreinte. C'est ce que garantit encoding/json en ordonnant les
// clés, et c'est la raison pour laquelle Fingerprint passe par JSON plutôt que par
// une représentation de structure.
func TestFingerprintIgnoresMapOrder(t *testing.T) {
	t.Parallel()

	left := map[string]int{"a": 1, "b": 2, "c": 3}
	right := map[string]int{"c": 3, "b": 2, "a": 1}

	if domain.Fingerprint(left) != domain.Fingerprint(right) {
		t.Error("l'ordre d'insertion d'une map ne doit pas changer l'empreinte")
	}
}

// TestFingerprintNeverEmpty : une empreinte vide vaudrait « pas d'empreinte », et
// domain.ErrIncomplete refuserait une requête pourtant valide.
func TestFingerprintNeverEmpty(t *testing.T) {
	t.Parallel()

	payloads := []any{nil, "", 0, false, map[string]any{}, []int{}, struct{}{}}
	for _, payload := range payloads {
		if got := domain.Fingerprint(payload); got == "" {
			t.Errorf("empreinte vide pour %#v", payload)
		}
	}
}

// TestFingerprintHandlesUnserializablePayload : une charge non sérialisable est un
// défaut de programmation. Le contrat est de rendre quand même une empreinte
// utilisable plutôt que de propager une erreur sur un chemin défensif.
func TestFingerprintHandlesUnserializablePayload(t *testing.T) {
	t.Parallel()

	type withChannel struct {
		Ch chan int
	}
	first := domain.Fingerprint(withChannel{Ch: nil})
	if first == "" {
		t.Fatal("empreinte vide pour une charge non sérialisable")
	}
	if second := domain.Fingerprint(withChannel{Ch: nil}); second != first {
		t.Errorf("empreinte de repli instable: %q puis %q", first, second)
	}
}
