package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintDistinguishesPayloads: a key reused with a different payload
// must be detected. If the fingerprints were confused, the second caller would
// receive the response of the first one — a data leak.
func TestFingerprintDistinguishesPayloads(t *testing.T) {
	t.Parallel()

	cases := map[string][2]any{
		"different value":   {map[string]int{"amount": 100}, map[string]int{"amount": 101}},
		"extra field":       {map[string]int{"amount": 100}, map[string]int{"amount": 100, "fee": 0}},
		"different type":    {map[string]any{"amount": 100}, map[string]any{"amount": "100"}},
		"nil against empty": {nil, map[string]any{}},
		"swapped values":    {map[string]int{"a": 1, "b": 2}, map[string]int{"a": 2, "b": 1}},
	}

	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			left, right := domain.Fingerprint(pair[0]), domain.Fingerprint(pair[1])
			if left == right {
				t.Errorf("identical fingerprints for two different payloads: %q", left)
			}
		})
	}
}
