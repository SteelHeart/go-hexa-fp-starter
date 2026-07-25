// Package fp fournit les primitives fonctionnelles qui ne concernent pas la
// gestion d'erreur : Option, composition et opérations sur les tranches.
//
// Comme result, ce paquet n'importe rien et ne doit jamais rien importer.
// Voir .arch-go.yml.
package fp

// Option porte une valeur éventuellement absente. Elle remplace un pointeur nil
// dans le domaine : l'absence devient un cas que le compilateur oblige à traiter.
//
// Sa valeur zéro est None.
type Option[T any] struct {
	value   T
	present bool
}

// Some construit une Option contenant une valeur.
func Some[T any](value T) Option[T] { return Option[T]{value: value, present: true} }

// None construit une Option vide.
func None[T any]() Option[T] { return Option[T]{} }

// IsSome indique si l'Option contient une valeur.
func (o Option[T]) IsSome() bool { return o.present }

// IsNone indique si l'Option est vide.
func (o Option[T]) IsNone() bool { return !o.present }

// Get expose la valeur et sa présence. Le booléen force le site d'appel à
// traiter l'absence.
func (o Option[T]) Get() (T, bool) { return o.value, o.present }

// ValueOr retourne la valeur contenue, ou la valeur de repli si l'Option est vide.
func (o Option[T]) ValueOr(fallback T) T {
	if o.present {
		return o.value
	}
	return fallback
}

// MapOption applique f à la valeur contenue. Une Option vide traverse inchangée.
func MapOption[T any, U any](o Option[T], f func(T) U) Option[U] {
	if !o.present {
		return None[U]()
	}
	return Some(f(o.value))
}

// FlatMapOption enchaîne une opération qui peut elle-même ne rien retourner.
func FlatMapOption[T any, U any](o Option[T], f func(T) Option[U]) Option[U] {
	if !o.present {
		return None[U]()
	}
	return f(o.value)
}

// FoldOption réduit les deux branches à une seule valeur.
func FoldOption[T any, R any](o Option[T], onSome func(T) R, onNone func() R) R {
	if o.present {
		return onSome(o.value)
	}
	return onNone()
}

// FromPointer convertit un pointeur en Option, sans déréférencer nil.
// C'est le point de conversion des frontières : au-delà, le domaine ne
// manipule plus de pointeur.
func FromPointer[T any](p *T) Option[T] {
	if p == nil {
		return None[T]()
	}
	return Some(*p)
}

// Identity retourne son argument. Utile comme branche neutre d'un Fold.
func Identity[T any](value T) T { return value }

// Pipe2 compose deux fonctions de gauche à droite.
func Pipe2[A any, B any, C any](f func(A) B, g func(B) C) func(A) C {
	return func(a A) C { return g(f(a)) }
}

// Pipe3 compose trois fonctions de gauche à droite.
func Pipe3[A any, B any, C any, D any](f func(A) B, g func(B) C, h func(C) D) func(A) D {
	return func(a A) D { return h(g(f(a))) }
}

// Map applique f à chaque élément et retourne une nouvelle tranche.
// L'entrée n'est jamais modifiée.
func Map[T any, U any](items []T, f func(T) U) []U {
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, f(item))
	}
	return out
}

// Filter retourne les éléments qui satisfont le prédicat.
func Filter[T any](items []T, keep func(T) bool) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

// Reduce replie la tranche sur un accumulateur.
func Reduce[T any, A any](items []T, initial A, f func(A, T) A) A {
	acc := initial
	for _, item := range items {
		acc = f(acc, item)
	}
	return acc
}

// Find retourne le premier élément satisfaisant le prédicat.
func Find[T any](items []T, match func(T) bool) Option[T] {
	for _, item := range items {
		if match(item) {
			return Some(item)
		}
	}
	return None[T]()
}
