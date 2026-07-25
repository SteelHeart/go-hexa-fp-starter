package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestIdentityIsNeutral : Identity rend son argument, et sert de branche neutre.
//
// Elle a l'air inutile jusqu'au premier Fold dont une branche n'a rien à faire.
// L'écrire à la main à chaque fois produirait des closures anonymes qui se
// ressemblent toutes et qu'on finit par confondre.
func TestIdentityIsNeutral(t *testing.T) {
	t.Parallel()

	if got := fp.Identity(42); got != 42 {
		t.Errorf("Identity(42) = %d", got)
	}
	if got := fp.Identity("texte"); got != "texte" {
		t.Errorf("Identity(\"texte\") = %q", got)
	}

	// Composée avec elle-même ou avec une autre fonction, elle reste neutre.
	compose := fp.Pipe2(fp.Identity[int], double)
	if got := compose(5); got != 10 {
		t.Errorf("Identity dans une composition = %d, attendu 10", got)
	}
}
