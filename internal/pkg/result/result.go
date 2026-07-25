// Package result fournit Result[T, E] : une valeur qui est soit un succès,
// soit une erreur — jamais les deux, jamais nil.
//
// Ce paquet n'importe rien, et ne doit jamais rien importer : c'est vérifié par
// arch-go.yml. Il est la fondation du cœur métier, qui doit rester pur.
//
// # Pourquoi des fonctions libres plutôt que des méthodes
//
// Go n'autorise pas de paramètre de type sur une méthode :
//
//	func (r Result[T, E]) Map[U any](f func(T) U) Result[U, E]  // ILLÉGAL
//	func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E]  // légal
//
// Toute transformation qui change le type est donc une fonction libre, et le
// chaînage fluide est impossible. Voir documentation/adr/002.
package result

// Result porte soit une valeur de succès, soit une erreur.
//
// Sa valeur zéro est un Err : un Result oublié échoue, il ne réussit pas
// silencieusement. C'est « deny par défaut » jusque dans le typage.
type Result[T any, E any] struct {
	value T
	err   E
	ok    bool
}

// Ok construit un Result en succès.
func Ok[T, E any](value T) Result[T, E] {
	return Result[T, E]{value: value, ok: true}
}

// Err construit un Result en erreur.
func Err[T, E any](err E) Result[T, E] {
	return Result[T, E]{err: err}
}

// IsOk indique si le Result porte une valeur de succès.
func (r Result[T, E]) IsOk() bool { return r.ok }

// IsErr indique si le Result porte une erreur.
func (r Result[T, E]) IsErr() bool { return !r.ok }

// Get expose les deux branches. Le booléen force le site d'appel à discriminer :
// c'est le seul moyen de sortir de la boîte.
//
// Les trois retours sont NOMMÉS parce que `(T, E, bool)` ne dit pas lequel est
// valide : `ok` vrai signifie que `value` porte le résultat et que `failure` est
// la valeur zéro, faux signifie l'inverse. Sans les noms, l'ordre est la seule
// documentation, et l'inverser resterait compilable.
func (r Result[T, E]) Get() (value T, failure E, ok bool) {
	return r.value, r.err, r.ok
}

// ValueOr retourne la valeur de succès, ou la valeur de repli si le Result est
// en erreur.
func (r Result[T, E]) ValueOr(fallback T) T {
	if r.ok {
		return r.value
	}
	return fallback
}

// Map applique f à la valeur de succès. Un Result en erreur traverse inchangé.
func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return Ok[U, E](f(r.value))
}

// MapErr applique f à l'erreur. Un Result en succès traverse inchangé.
//
// C'est la fonction de traduction des frontières : un adaptateur secondaire
// l'utilise pour convertir une erreur technique en erreur de domaine.
func MapErr[T, E, F any](r Result[T, E], f func(E) F) Result[T, F] {
	if r.ok {
		return Ok[T, F](r.value)
	}
	return Err[T, F](f(r.err))
}

// FlatMap enchaîne une opération qui peut elle-même échouer, en court-circuitant
// au premier Err.
func FlatMap[T, U, E any](r Result[T, E], f func(T) Result[U, E]) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return f(r.value)
}

// Fold réduit les deux branches à une seule valeur. C'est la sortie canonique
// d'un Result dans un adaptateur primaire.
func Fold[T, E, R any](r Result[T, E], onOk func(T) R, onErr func(E) R) R {
	if r.ok {
		return onOk(r.value)
	}
	return onErr(r.err)
}

// Chain compose des étapes de même type, en court-circuitant au premier Err.
//
// C'est le patron imposé pour écrire un cas d'usage : sans do-notation, une
// suite d'étapes homogènes se lit infiniment mieux qu'une pyramide de FlatMap.
func Chain[T, E any](initial Result[T, E], steps ...func(T) Result[T, E]) Result[T, E] {
	acc := initial
	for _, step := range steps {
		if !acc.ok {
			return acc
		}
		acc = step(acc.value)
	}
	return acc
}

// Tap exécute un effet sur la valeur de succès et retourne le Result inchangé.
// Réservé aux décorateurs : le cœur n'a pas d'effet à produire.
func Tap[T, E any](r Result[T, E], f func(T)) Result[T, E] {
	if r.ok {
		f(r.value)
	}
	return r
}

// TapErr exécute un effet sur l'erreur et retourne le Result inchangé.
func TapErr[T, E any](r Result[T, E], f func(E)) Result[T, E] {
	if !r.ok {
		f(r.err)
	}
	return r
}

// OrElse remplace une erreur par un Result de repli.
func OrElse[T, E any](r Result[T, E], f func(E) Result[T, E]) Result[T, E] {
	if r.ok {
		return r
	}
	return f(r.err)
}

// Collect transforme une liste de Result en un Result de liste, en s'arrêtant à
// la première erreur.
func Collect[T, E any](results []Result[T, E]) Result[[]T, E] {
	values := make([]T, 0, len(results))
	for _, r := range results {
		if !r.ok {
			return Err[[]T, E](r.err)
		}
		values = append(values, r.value)
	}
	return Ok[[]T, E](values)
}

// Traverse applique f à chaque élément et rassemble les résultats, en
// s'arrêtant à la première erreur.
func Traverse[T, U, E any](items []T, f func(T) Result[U, E]) Result[[]U, E] {
	values := make([]U, 0, len(items))
	for _, item := range items {
		r := f(item)
		if !r.ok {
			return Err[[]U, E](r.err)
		}
		values = append(values, r.value)
	}
	return Ok[[]U, E](values)
}
