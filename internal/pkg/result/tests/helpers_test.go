// Package tests holds the BLACK BOX tests of the result primitive: they only use
// the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
//
// # Why the algebraic laws are tested here
//
// `Result` is not an ordinary data structure: it is the shape imposed on EVERY
// use case in the repository. If `Map` did not preserve identity, or if
// `FlatMap` were not associative, then refactoring a pipeline — merging two
// steps, extracting a third — would change the behaviour of the application.
// The laws are what makes that refactoring safe, and they are not visible when
// reading the code.
package tests

import (
	"strconv"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// failure is a test error, deliberately NOT an `error`: Result[T, E] accepts any
// error type, and the business core puts its own taxonomy there rather than the
// standard interface.
type failure string

// Three pure functions, to compose with.
func double(n int) int    { return n * 2 }
func toText(n int) string { return strconv.Itoa(n) }
func increment(n int) int { return n + 1 }

// okInt and errInt shorten the most frequent constructions.
func okInt(n int) result.Result[int, failure] { return result.Ok[int, failure](n) }

func errInt(e failure) result.Result[int, failure] { return result.Err[int, failure](e) }

// valueOf extracts the success value, or the zero value of T.
func valueOf[T, E any](r result.Result[T, E]) T {
	value, _, _ := r.Get()
	return value
}

// causeOf extracts the error, or its zero value.
func causeOf[T, E any](r result.Result[T, E]) E {
	_, err, _ := r.Get()
	return err
}
