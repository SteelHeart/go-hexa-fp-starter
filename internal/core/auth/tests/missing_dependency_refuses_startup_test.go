package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestMissingDependencyRefusesStartup fails the wiring, not production.
//
// Without this refusal, a forgotten dependency would produce a nil pointer
// dereference on the FIRST real sign-in — so in production, and with a stack
// trace that does not say which one was missing. The module refuses at startup
// and NAMES the missing port: that is the difference between thirty seconds and
// an evening.
func TestMissingDependencyRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	complete := deps(c)

	cases := map[string]auth.Deps{
		"HashSecret":   {VerifySecret: complete.VerifySecret, Now: complete.Now},
		"VerifySecret": {HashSecret: complete.HashSecret, Now: complete.Now},
		"Now":          {HashSecret: complete.HashSecret, VerifySecret: complete.VerifySecret},
		"everything":   {},
	}

	for name, incomplete := range cases {
		_, err := auth.New(config.Module{Enabled: true, Driver: "memory"}, incomplete)
		if !errors.Is(err, auth.ErrMissingDependency) {
			t.Errorf("without %s: want ErrMissingDependency, got %v", name, err)
		}
	}
}

// TestDisabledModuleNeedsNoDependency: a turned-off module mounts with nothing.
//
// Demanding its dependencies would fail the server startup because of a module
// nobody enabled — and would push people to wire dummy dependencies "so that it
// passes", which is exactly how a disabled module ends up being enabled without
// anyone noticing.
func TestDisabledModuleNeedsNoDependency(t *testing.T) {
	t.Parallel()

	if _, err := auth.New(config.Module{Enabled: false}, auth.Deps{}); err != nil {
		t.Fatalf("a disabled module must demand no dependency: %v", err)
	}
}
