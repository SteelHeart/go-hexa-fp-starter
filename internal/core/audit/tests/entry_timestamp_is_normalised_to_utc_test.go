package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// TestEntryTimestampIsNormalisedToUTC : un journal d'audit relu six mois plus tard
// depuis un autre fuseau doit se comparer sans arithmétique. WithTime normalise.
func TestEntryTimestampIsNormalisedToUTC(t *testing.T) {
	t.Parallel()

	local := time.FixedZone("UTC+3", 3*60*60)
	at := time.Date(2026, time.July, 25, 15, 30, 0, 0, local)

	stamped := domain.Entry{}.WithTime(at)
	if stamped.At.Location() != time.UTC {
		t.Errorf("fuseau = %v, attendu UTC", stamped.At.Location())
	}
	if !stamped.At.Equal(at) {
		t.Errorf("l'instant a été déplacé: %v ≠ %v", stamped.At, at)
	}
}
