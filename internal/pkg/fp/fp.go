// Package fp provides the functional primitives that are not about error
// handling: Option, composition and slice operations.
//
// Like result, this package imports nothing and must never import anything.
// See arch-go.yml.
package fp

// Option carries a possibly absent value. It replaces a nil pointer inside the
// domain: absence becomes a case the compiler forces you to handle.
//
// Its zero value is None.
type Option[T any] struct {
	value   T
	present bool
}

// Some builds an Option holding a value.
func Some[T any](value T) Option[T] { return Option[T]{value: value, present: true} }

// None builds an empty Option.
func None[T any]() Option[T] { return Option[T]{} }

// IsSome reports whether the Option holds a value.
func (o Option[T]) IsSome() bool { return o.present }

// IsNone reports whether the Option is empty.
func (o Option[T]) IsNone() bool { return !o.present }

// Get exposes the value and its presence. The boolean forces the call site to
// handle absence.
func (o Option[T]) Get() (T, bool) { return o.value, o.present }

// ValueOr returns the held value, or the fallback if the Option is empty.
func (o Option[T]) ValueOr(fallback T) T {
	if o.present {
		return o.value
	}
	return fallback
}

// MapOption applies f to the held value. An empty Option passes through unchanged.
func MapOption[T, U any](o Option[T], f func(T) U) Option[U] {
	if !o.present {
		return None[U]()
	}
	return Some(f(o.value))
}

// FlatMapOption chains an operation that may itself return nothing.
func FlatMapOption[T, U any](o Option[T], f func(T) Option[U]) Option[U] {
	if !o.present {
		return None[U]()
	}
	return f(o.value)
}

// FoldOption reduces both branches to a single value.
func FoldOption[T, R any](o Option[T], onSome func(T) R, onNone func() R) R {
	if o.present {
		return onSome(o.value)
	}
	return onNone()
}

// FromPointer converts a pointer into an Option, without dereferencing nil.
// This is the boundary conversion point: past it, the domain no longer handles
// pointers.
func FromPointer[T any](p *T) Option[T] {
	if p == nil {
		return None[T]()
	}
	return Some(*p)
}

// Identity returns its argument. Useful as the neutral branch of a Fold.
func Identity[T any](value T) T { return value }

// Pipe2 composes two functions left to right.
func Pipe2[A, B, C any](f func(A) B, g func(B) C) func(A) C {
	return func(a A) C { return g(f(a)) }
}

// Pipe3 composes three functions left to right.
func Pipe3[A, B, C, D any](f func(A) B, g func(B) C, h func(C) D) func(A) D {
	return func(a A) D { return h(g(f(a))) }
}

// Map applies f to every item and returns a new slice.
// The input is never modified.
func Map[T, U any](items []T, f func(T) U) []U {
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, f(item))
	}
	return out
}

// Filter returns the items satisfying the predicate.
func Filter[T any](items []T, keep func(T) bool) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

// Reduce folds the slice onto an accumulator.
func Reduce[T, A any](items []T, initial A, f func(A, T) A) A {
	acc := initial
	for _, item := range items {
		acc = f(acc, item)
	}
	return acc
}

// Find returns the first item satisfying the predicate.
func Find[T any](items []T, match func(T) bool) Option[T] {
	for _, item := range items {
		if match(item) {
			return Some(item)
		}
	}
	return None[T]()
}
