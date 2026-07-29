package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDevelopmentIsTheOnlyEnvironmentWithoutHSTS: deny by default on HSTS.
//
// # The defect this test catches
//
// The choice is made on `env.IsDevelopment()`. Writing `!env.IsProduction()`
// instead looks equivalent and is nothing of the sort: UAT and test would lose
// HSTS. UAT is an exposed environment, often with realistic data, and nobody
// looks at the response headers there.
//
// The test also covers an UNKNOWN environment — therefore misconfigured. It
// must receive the FULL hardening: a configuration we have not managed to read
// is exactly the moment when one has to protect oneself the most, not the least.
//
// The waiver must name itself. That is why `SecurityHeaders()` protects and why
// `SecurityHeadersWithoutHSTS()` carries its waiver in its name.
func TestDevelopmentIsTheOnlyEnvironmentWithoutHSTS(t *testing.T) {
	t.Parallel()

	const header = "Strict-Transport-Security"

	// Plain-text development, and it alone, gives up HSTS: on
	// `http://localhost`, HSTS would write into the browser a requirement for
	// HTTPS that the workstation cannot satisfy.
	if got := get(t, config.EnvDevelopment, nil, "/healthz").Header().Get(header); got != "" {
		t.Errorf("development carries HSTS (%q) — the developer would lose access to their server", got)
	}

	for _, env := range []config.Environment{
		config.EnvTest,
		config.EnvUAT,
		config.EnvProduction,
		"an-unknown-environment",
		"",
	} {
		if got := get(t, env, nil, "/healthz").Header().Get(header); got == "" {
			t.Errorf("env=%q WITHOUT HSTS — only development may give it up, and it must name itself", env)
		}
	}
}

// TestHardeningHeadersAreOnEveryEnvironment: the rest of the hardening is unconditional.
//
// Only HSTS depends on the environment. If `securityHeadersFor` picked the
// wrong constructor, this test would see ALL the headers disappear, not only
// HSTS — and that is the silent failure we want to tell apart from a mere HSTS
// oversight.
func TestHardeningHeadersAreOnEveryEnvironment(t *testing.T) {
	t.Parallel()

	expected := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
		"Content-Security-Policy":    "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
	}

	for _, env := range []config.Environment{
		config.EnvDevelopment, config.EnvTest, config.EnvUAT, config.EnvProduction,
	} {
		headers := get(t, env, nil, "/healthz").Header()
		for name, want := range expected {
			if got := headers.Get(name); got != want {
				t.Errorf("env=%q: %s = %q, want %q", env, name, got, want)
			}
		}
	}
}
