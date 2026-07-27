package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestInvalidCursorRejectsTheWholeRequest : un curseur invalide fait échouer la
// demande ENTIÈRE.
//
// La tentation serait de garder la limite et de repartir de la première page. Ce
// serait un repli silencieux sur une entrée corrompue : le client croirait avancer
// dans une liste alors qu'il en relit sans cesse le début.
func TestInvalidCursorRejectsTheWholeRequest(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("!!!pas du base64!!!", 50)
	if !errors.Is(err, pagination.ErrInvalidCursor) {
		t.Fatalf("erreur = %v, attendu ErrInvalidCursor", err)
	}
	if req.Limit != 0 || req.HasAfter {
		t.Errorf("la demande doit être rendue VIDE en cas d'erreur, reçu %+v", req)
	}
}
