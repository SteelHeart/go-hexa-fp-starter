// Package tests holds the BLACK BOX tests of the registration domain: they only
// use the public API, exactly like a use case would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
//
// # Why these tests are the most profitable in the repository
//
// The domain is PURE: no I/O, no clock, no randomness. It therefore tests in
// microseconds, without a container, without a double, without any assembly.
// That is where every business rule and every edge case must be covered — not
// higher up, where each test costs a hundred times more to prove the same thing.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// validEmail builds an address known to pass, or fails the test. Avoids
// repeating the discrimination of the Result in every file.
func validEmail(t *testing.T, raw string) domain.Email {
	t.Helper()
	value, err, ok := domain.NewEmail(raw).Get()
	if !ok {
		t.Fatalf("NewEmail(%q) should have succeeded, refused: %v", raw, err)
	}
	return value
}

// codeOf extracts the error code of a failed Result.
func codeOf[T any](t *testing.T, r result.Result[T, domain.Error]) domain.ErrorCode {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("a failure was expected, got a success")
	}
	return err.Code
}

// failureOf extracts the complete error of a failed Result.
func failureOf[T any](t *testing.T, r result.Result[T, domain.Error]) domain.Error {
	t.Helper()
	_, err, ok := r.Get()
	if ok {
		t.Fatal("a failure was expected, got a success")
	}
	return err
}
