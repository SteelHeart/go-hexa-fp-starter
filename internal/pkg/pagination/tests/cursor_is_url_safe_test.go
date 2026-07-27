package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestCursorIsURLSafe : un curseur voyage dans une query string.
//
// Un `+`, un `/` ou un `=` y seraient réencodés par un client, un cache ou un
// proxy — et le curseur reviendrait altéré. L'encodage base64 URL sans remplissage
// évite les trois. Le défaut ne se verrait qu'en production, sur certains clients
// seulement, ce qui en fait un excellent candidat au test automatisé.
func TestCursorIsURLSafe(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"a", "user-42", "01J8ZQ9V3K4M5N6P7R8S9T0V1W", "é#?&=/+"} {
		encoded := pagination.Cursor{CreatedAt: base(), ID: id}.Encode()
		if strings.ContainsAny(encoded, "+/=") {
			t.Errorf("curseur %q contient un caractère réencodé en URL", encoded)
		}
	}
}
