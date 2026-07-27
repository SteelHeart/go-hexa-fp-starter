package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTapObservesWithoutAltering : Tap et TapErr produisent un effet et rendent le
// Result INCHANGÉ.
//
// Elles sont réservées aux décorateurs — tracer, journaliser, compter. Si elles
// pouvaient modifier le Result, un décorateur d'observabilité changerait le
// comportement de l'application qu'il observe, et le défaut serait indétectable :
// personne ne soupçonne le traceur.
func TestTapObservesWithoutAltering(t *testing.T) {
	t.Parallel()

	var vues []string

	succes := result.Tap(okInt(7), func(n int) { vues = append(vues, "ok:"+versTexte(n)) })
	succes = result.TapErr(succes, func(erreur) { vues = append(vues, "err") })

	if valeur(succes) != 7 || !succes.IsOk() {
		t.Error("Tap ne doit pas altérer un succès")
	}

	echec := result.Tap(errInt("refusé"), func(int) { vues = append(vues, "ok") })
	echec = result.TapErr(echec, func(e erreur) { vues = append(vues, "err:"+string(e)) })

	if echec.IsOk() || cause(echec) != "refusé" {
		t.Error("TapErr ne doit pas altérer une erreur")
	}

	if len(vues) != 2 || vues[0] != "ok:7" || vues[1] != "err:refusé" {
		t.Errorf("effets observés = %v, attendu un par branche", vues)
	}
}
