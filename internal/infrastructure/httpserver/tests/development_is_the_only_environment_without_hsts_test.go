package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestDevelopmentIsTheOnlyEnvironmentWithoutHSTS : deny par défaut sur HSTS.
//
// # Le défaut que ce test attrape
//
// Le choix se fait sur `env.IsDevelopment()`. Écrire `!env.IsProduction()` à la
// place semble équivalent et ne l'est pas du tout : UAT et test perdraient HSTS.
// UAT est un environnement exposé, souvent avec des données réalistes, et
// personne n'y regarde les en-têtes de réponse.
//
// Le test couvre aussi un environnement INCONNU — donc mal configuré. Il doit
// recevoir le durcissement COMPLET : une configuration qu'on n'a pas su lire est
// exactement le moment où il faut se protéger le plus, pas le moins.
//
// La renonciation doit se nommer. C'est pour ça que `SecurityHeaders()` protège
// et que `SecurityHeadersWithoutHSTS()` porte son renoncement dans son nom.
func TestDevelopmentIsTheOnlyEnvironmentWithoutHSTS(t *testing.T) {
	t.Parallel()

	const header = "Strict-Transport-Security"

	// Le développement en clair, et lui seul, renonce à HSTS : sur
	// `http://localhost`, HSTS inscrirait dans le navigateur une exigence de
	// HTTPS que le poste ne peut pas satisfaire.
	if got := get(t, config.EnvDevelopment, nil, "/healthz").Header().Get(header); got != "" {
		t.Errorf("development porte HSTS (%q) — le développeur perdrait l'accès à son serveur", got)
	}

	for _, env := range []config.Environment{
		config.EnvTest,
		config.EnvUAT,
		config.EnvProduction,
		"un-environnement-inconnu",
		"",
	} {
		if got := get(t, env, nil, "/healthz").Header().Get(header); got == "" {
			t.Errorf("env=%q SANS HSTS — seul development doit y renoncer, et il doit se nommer", env)
		}
	}
}

// TestHardeningHeadersAreOnEveryEnvironment : le reste du durcissement est inconditionnel.
//
// Seul HSTS dépend de l'environnement. Si `securityHeadersFor` se trompait de
// constructeur, ce test verrait disparaître TOUS les en-têtes, pas seulement HSTS
// — et c'est la panne silencieuse qu'on veut distinguer d'un simple oubli de HSTS.
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
				t.Errorf("env=%q: %s = %q, attendu %q", env, name, got, want)
			}
		}
	}
}
