package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapErrTranslatesBoundaries: MapErr is the boundary translation function.
//
// A secondary adapter uses it to convert a technical error — an SQL constraint
// violation — into a domain error — "this address is already taken". Without it,
// either the domain would know SQLSTATE codes, or the caller would receive an
// error it cannot interpret.
//
// Required symmetry: it must NOT touch a success.
func TestMapErrTranslatesBoundaries(t *testing.T) {
	t.Parallel()

	translate := func(e failure) string { return "domain: " + string(e) }

	failed := result.MapErr(errInt("23505"), translate)
	if failed.IsOk() {
		t.Fatal("MapErr on an error must return an error")
	}
	if causeOf(failed) != "domain: 23505" {
		t.Errorf("translated error = %q", causeOf(failed))
	}

	called := false
	succeeded := result.MapErr(okInt(7), func(e failure) string {
		called = true
		return string(e)
	})
	if called {
		t.Error("the translation must NOT be called on a success")
	}
	if valueOf(succeeded) != 7 {
		t.Errorf("the value must pass through unchanged, got %d", valueOf(succeeded))
	}
}
