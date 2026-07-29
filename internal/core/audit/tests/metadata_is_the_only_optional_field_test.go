package tests

import (
	"context"
	"testing"
)

// TestMetadataIsTheOnlyOptionalField: an audit fact with no additional context
// stays usable. Requiring metadata would push people to invent some, and the
// first invented field would be a personal datum copied over "just in case".
func TestMetadataIsTheOnlyOptionalField(t *testing.T) {
	t.Parallel()

	mod, _ := newLogModule(t)
	entry := completeEntry()
	entry.Metadata = nil

	if err := mod.Record(context.Background(), entry); err != nil {
		t.Errorf("an entry without metadata must be accepted: %v", err)
	}
}
