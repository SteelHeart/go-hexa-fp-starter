package tests

import (
	"context"
	"testing"
)

// TestDeleteIsIdempotent : supprimer un objet absent n'est pas une erreur.
//
// L'appelant veut que l'objet ne soit plus là, et il n'y est plus. Faire échouer
// une suppression idempotente obligerait chaque appelant à distinguer deux cas
// équivalents — et le premier qui oublierait la distinction propagerait une erreur
// pour un succès.
func TestDeleteIsIdempotent(t *testing.T) {
	t.Parallel()

	mod := newDiskModule(t)
	ctx := context.Background()

	located, err := mod.Put(ctx, object("temporaire.txt", "contenu"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := mod.Delete(ctx, located.Key); err != nil {
		t.Fatalf("première suppression: %v", err)
	}
	if err := mod.Delete(ctx, located.Key); err != nil {
		t.Errorf("seconde suppression = %v, attendu nil", err)
	}

	if _, err := mod.Get(ctx, located.Key); err == nil {
		t.Error("l'objet supprimé ne doit plus être lisible")
	}
}
