// Package result provides Result[T, E]: a value that is either a success or an
// error — never both, never nil.
//
// This package imports nothing, and must never import anything: arch-go.yml
// enforces it. It is the foundation of the business core, which must stay pure.
//
// # Why free functions rather than methods
//
// Go does not allow a type parameter on a method:
//
//	func (r Result[T, E]) Map[U any](f func(T) U) Result[U, E]  // ILLEGAL
//	func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E]  // legal
//
// Any transformation that changes the type is therefore a free function, and
// fluent chaining is impossible. See documentation/adr/002.
package result

// Result carries either a success value or an error.
//
// Its zero value is an Err: a forgotten Result fails, it does not silently
// succeed. That is "deny by default", all the way into the type system.
type Result[T any, E any] struct {
	value T
	err   E
	ok    bool
}

// Ok builds a successful Result.
func Ok[T, E any](value T) Result[T, E] {
	return Result[T, E]{value: value, ok: true}
}

// Err builds a failed Result.
func Err[T, E any](err E) Result[T, E] {
	return Result[T, E]{err: err}
}

// IsOk reports whether the Result carries a success value.
func (r Result[T, E]) IsOk() bool { return r.ok }

// IsErr reports whether the Result carries an error.
func (r Result[T, E]) IsErr() bool { return !r.ok }

// Get exposes both branches. The boolean forces the call site to discriminate:
// it is the only way out of the box.
//
// The three returns are NAMED because `(T, E, bool)` does not say which one is
// valid: a true `ok` means `value` holds the result and `failure` is the zero
// value, false means the opposite. Without the names, ordering is the only
// documentation, and swapping it would still compile.
func (r Result[T, E]) Get() (value T, failure E, ok bool) {
	return r.value, r.err, r.ok
}

// ValueOr returns the success value, or the fallback if the Result failed.
func (r Result[T, E]) ValueOr(fallback T) T {
	if r.ok {
		return r.value
	}
	return fallback
}

// Map applies f to the success value. A failed Result passes through unchanged.
func Map[T, U, E any](r Result[T, E], f func(T) U) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return Ok[U, E](f(r.value))
}

// MapErr applies f to the error. A successful Result passes through unchanged.
//
// This is the boundary translation function: a secondary adapter uses it to
// convert a technical error into a domain error.
func MapErr[T, E, F any](r Result[T, E], f func(E) F) Result[T, F] {
	if r.ok {
		return Ok[T, F](r.value)
	}
	return Err[T, F](f(r.err))
}

// FlatMap chains an operation that may itself fail, short-circuiting on the
// first Err.
func FlatMap[T, U, E any](r Result[T, E], f func(T) Result[U, E]) Result[U, E] {
	if !r.ok {
		return Err[U, E](r.err)
	}
	return f(r.value)
}

// Fold reduces both branches to a single value. It is the canonical way out of
// a Result inside a primary adapter.
func Fold[T, E, R any](r Result[T, E], onOk func(T) R, onErr func(E) R) R {
	if r.ok {
		return onOk(r.value)
	}
	return onErr(r.err)
}

// Chain composes steps of the same type, short-circuiting on the first Err.
//
// This is the mandated shape for writing a use case: without do-notation, a
// sequence of homogeneous steps reads infinitely better than a pyramid of
// FlatMap calls.
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

// Tap runs an effect on the success value and returns the Result unchanged.
// Reserved for decorators: the core has no effect to produce.
func Tap[T, E any](r Result[T, E], f func(T)) Result[T, E] {
	if r.ok {
		f(r.value)
	}
	return r
}

// TapErr runs an effect on the error and returns the Result unchanged.
func TapErr[T, E any](r Result[T, E], f func(E)) Result[T, E] {
	if !r.ok {
		f(r.err)
	}
	return r
}

// OrElse replaces an error with a fallback Result.
func OrElse[T, E any](r Result[T, E], f func(E) Result[T, E]) Result[T, E] {
	if r.ok {
		return r
	}
	return f(r.err)
}

// Collect turns a list of Results into a Result of list, stopping at the first
// error.
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

// Traverse applies f to every item and gathers the results, stopping at the
// first error.
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
