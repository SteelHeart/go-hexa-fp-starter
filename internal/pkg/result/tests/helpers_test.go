// Package tests contient les tests en BOÎTE NOIRE de la primitive result : ils
// n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
//
// # Pourquoi les lois algébriques sont testées ici
//
// `Result` n'est pas une structure de données ordinaire : c'est la forme imposée à
// TOUT cas d'usage du dépôt. Si `Map` ne préservait pas l'identité, ou si
// `FlatMap` n'était pas associatif, alors refactoriser un pipeline — regrouper
// deux étapes, en extraire une troisième — changerait le comportement de
// l'application. Les lois sont ce qui rend ce refactoring sûr, et elles ne se
// voient pas à la lecture du code.
package tests

import (
	"strconv"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// erreur est une erreur de test, volontairement PAS une `error` : Result[T, E]
// accepte n'importe quel type d'erreur, et le cœur métier y met sa propre
// taxonomie plutôt que l'interface standard.
type erreur string

// Trois fonctions pures, pour composer.
func double(n int) int       { return n * 2 }
func versTexte(n int) string { return strconv.Itoa(n) }
func incremente(n int) int   { return n + 1 }

// okInt et errInt raccourcissent les constructions les plus fréquentes.
func okInt(n int) result.Result[int, erreur] { return result.Ok[int, erreur](n) }

func errInt(e erreur) result.Result[int, erreur] { return result.Err[int, erreur](e) }

// valeur extrait la valeur de succès, ou la valeur zéro de T.
func valeur[T any, E any](r result.Result[T, E]) T {
	value, _, _ := r.Get()
	return value
}

// cause extrait l'erreur, ou sa valeur zéro.
func cause[T any, E any](r result.Result[T, E]) E {
	_, err, _ := r.Get()
	return err
}
