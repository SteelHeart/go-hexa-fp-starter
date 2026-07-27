package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestFlatMapObeysMonadLaws vérifie les trois lois qui rendent un pipeline
// réorganisable.
//
//  1. IDENTITÉ À GAUCHE : FlatMap(Ok(x), f) == f(x). Emballer une valeur pour la
//     déballer aussitôt ne doit rien changer.
//  2. IDENTITÉ À DROITE : FlatMap(r, Ok) == r. Ajouter une étape qui se contente
//     de réussir ne doit rien changer.
//  3. ASSOCIATIVITÉ : FlatMap(FlatMap(r, f), g) == FlatMap(r, x => FlatMap(f(x), g)).
//     C'est LA loi qui compte au quotidien : elle dit que le PARENTHÉSAGE des
//     étapes n'a aucune importance. Extraire trois étapes d'un cas d'usage vers une
//     fonction dédiée est donc sûr — sans elle, ce serait un changement de
//     comportement déguisé en réorganisation.
func TestFlatMapObeysMonadLaws(t *testing.T) {
	t.Parallel()

	f := func(n int) result.Result[int, erreur] {
		if n < 0 {
			return errInt("négatif")
		}
		return okInt(double(n))
	}
	g := func(n int) result.Result[int, erreur] {
		if n > 100 {
			return errInt("trop grand")
		}
		return okInt(incremente(n))
	}

	// 1. Identité à gauche.
	if valeur(result.FlatMap(okInt(5), f)) != valeur(f(5)) {
		t.Error("identité à gauche rompue")
	}

	// 2. Identité à droite, sur les deux branches.
	for _, r := range []result.Result[int, erreur]{okInt(5), errInt("refusé")} {
		neutre := result.FlatMap(r, result.Ok[int, erreur])
		if neutre.IsOk() != r.IsOk() || valeur(neutre) != valeur(r) || cause(neutre) != cause(r) {
			t.Error("identité à droite rompue")
		}
	}

	// 3. Associativité, y compris sur les entrées qui court-circuitent.
	for _, depart := range []result.Result[int, erreur]{okInt(5), okInt(-1), okInt(80), errInt("refusé")} {
		gauche := result.FlatMap(result.FlatMap(depart, f), g)
		droite := result.FlatMap(depart, func(n int) result.Result[int, erreur] {
			return result.FlatMap(f(n), g)
		})
		if gauche.IsOk() != droite.IsOk() ||
			valeur(gauche) != valeur(droite) ||
			cause(gauche) != cause(droite) {
			t.Errorf("associativité rompue au départ de %v", valeur(depart))
		}
	}
}
