package tests

import (
	"context"
	"io"
	"testing"
)

// TestPutThenGetReturnsTheContent : le chemin nominal, sans lequel aucune des
// garanties de sécurité n'aurait d'intérêt.
func TestPutThenGetReturnsTheContent(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()
	const content = "contenu du justificatif"

	located, err := mod.Put(ctx, object("justificatif.pdf", content))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if located.Key == "" {
		t.Fatal("Put doit rendre une clé")
	}

	reader, err := mod.Get(ctx, located.Key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = reader.Close() }()

	read, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if string(read) != content {
		t.Errorf("contenu relu = %q, attendu %q", read, content)
	}
}
