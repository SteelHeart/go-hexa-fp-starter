package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
)

// TestEntryTimestampIsNormalisedToUTC: an audit log re-read six months later
// from another time zone must compare without arithmetic. WithTime normalises.
func TestEntryTimestampIsNormalisedToUTC(t *testing.T) {
	t.Parallel()

	local := time.FixedZone("UTC+3", 3*60*60)
	at := time.Date(2026, time.July, 25, 15, 30, 0, 0, local)

	stamped := domain.Entry{}.WithTime(at)
	if stamped.At.Location() != time.UTC {
		t.Errorf("time zone = %v, want UTC", stamped.At.Location())
	}
	if !stamped.At.Equal(at) {
		t.Errorf("the instant has been moved: %v ≠ %v", stamped.At, at)
	}
}
