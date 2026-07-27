package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestOrElseRecoversOnlyFromErrors : OrElse remplace une erreur par un repli, et
// laisse un succès intact.
//
// C'est le seul moyen de se rattraper sans sortir de la boîte. Si elle touchait
// aussi les succès, elle deviendrait un « remplace toujours » — et le repli
// écraserait des valeurs légitimes sans que rien ne le signale.
func TestOrElseRecoversOnlyFromErrors(t *testing.T) {
	t.Parallel()

	repli := func(erreur) result.Result[int, erreur] { return okInt(0) }

	recupere := result.OrElse(errInt("cache indisponible"), repli)
	if !recupere.IsOk() {
		t.Fatal("OrElse doit remplacer une erreur par le repli")
	}
	if valeur(recupere) != 0 {
		t.Errorf("valeur de repli = %d, attendu 0", valeur(recupere))
	}

	appele := false
	intact := result.OrElse(okInt(7), func(erreur) result.Result[int, erreur] {
		appele = true
		return okInt(0)
	})
	if appele {
		t.Error("le repli ne doit PAS être appelé sur un succès")
	}
	if valeur(intact) != 7 {
		t.Errorf("valeur = %d, attendu 7 inchangé", valeur(intact))
	}
}
