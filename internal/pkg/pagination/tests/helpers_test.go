// Package tests contient les tests en BOÎTE NOIRE de la pagination par curseur :
// ils n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import (
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// ligne est une entité paginable de test.
type ligne struct {
	ID        string
	CreatedAt time.Time
}

// base est l'instant de référence.
func base() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// curseurDe extrait le curseur d'une ligne.
func curseurDe(l ligne) pagination.Cursor {
	return pagination.Cursor{CreatedAt: l.CreatedAt, ID: l.ID}
}

// lignes fabrique n lignes espacées d'une seconde, dans l'ordre.
func lignes(n int) []ligne {
	out := make([]ligne, 0, n)
	for i := range n {
		out = append(out, ligne{
			ID:        string(rune('a' + i)),
			CreatedAt: base().Add(time.Duration(i) * time.Second),
		})
	}
	return out
}
