// Package tests holds the BLACK BOX tests of the functional primitives: they
// only use the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import "strconv"

// Three pure functions, to compose with.
func double(n int) int    { return n * 2 }
func increment(n int) int { return n + 1 }
func toText(n int) string { return strconv.Itoa(n) }

// even is the reference predicate of the slice tests.
func even(n int) bool { return n%2 == 0 }
