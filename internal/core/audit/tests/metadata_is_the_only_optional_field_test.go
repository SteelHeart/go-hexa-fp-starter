package tests

import (
	"context"
	"testing"
)

// TestMetadataIsTheOnlyOptionalField : un fait d'audit sans contexte supplémentaire
// reste exploitable. Exiger des métadonnées pousserait à en inventer, et le premier
// champ inventé serait une donnée personnelle recopiée « au cas où ».
func TestMetadataIsTheOnlyOptionalField(t *testing.T) {
	t.Parallel()

	mod, _ := newLogModule(t)
	entry := completeEntry()
	entry.Metadata = nil

	if err := mod.Record(context.Background(), entry); err != nil {
		t.Errorf("une entrée sans métadonnées doit être acceptée: %v", err)
	}
}
