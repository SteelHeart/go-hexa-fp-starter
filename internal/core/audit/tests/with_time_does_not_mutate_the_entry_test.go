package tests

import (
	"testing"
)

// TestWithTimeDoesNotMutateTheEntry : domain.Entry est une valeur. Une méthode qui
// muterait son receveur rendrait deux enregistrements du même fait divergents.
func TestWithTimeDoesNotMutateTheEntry(t *testing.T) {
	t.Parallel()

	original := completeEntry()
	_ = original.WithTime(recordedAt())

	if !original.At.IsZero() {
		t.Errorf("WithTime a muté l'entrée d'origine: At = %v", original.At)
	}
}
