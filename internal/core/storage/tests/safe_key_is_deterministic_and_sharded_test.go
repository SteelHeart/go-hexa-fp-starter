package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestSafeKeyIsDeterministicAndSharded : deux appels sur le même nom donnent la
// même clé, et la clé est répartie sur deux niveaux de répertoires.
//
// La répartition n'est pas cosmétique : un répertoire plat à cent mille entrées
// devient impraticable sur la plupart des systèmes de fichiers, et le magasin
// ralentit d'un coup, tard, sans qu'on comprenne pourquoi.
func TestSafeKeyIsDeterministicAndSharded(t *testing.T) {
	t.Parallel()

	first, err := domain.SafeKey("facture-2026-07.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	second, err := domain.SafeKey("facture-2026-07.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if first != second {
		t.Errorf("clé instable: %q puis %q", first, second)
	}

	parts := strings.Split(first.String(), "/")
	if len(parts) != 3 {
		t.Fatalf("clé = %q, attendu trois niveaux", first)
	}
	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		t.Errorf("répartition = %q/%q, attendu deux niveaux de 2 caractères", parts[0], parts[1])
	}
}
