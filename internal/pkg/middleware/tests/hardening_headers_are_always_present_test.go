package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestHardeningHeadersAreAlwaysPresent : les en-têtes de durcissement sont posés
// sur TOUTE réponse, HSTS compris par défaut.
//
// Chacun ferme une attaque précise, et aucun ne se voit quand il manque :
//
//   - `X-Content-Type-Options: nosniff` — sans lui, un navigateur peut décider
//     qu'un fichier téléversé est du HTML et l'exécuter dans le contexte du site.
//   - `X-Frame-Options: DENY` — empêche le détournement de clic.
//   - `Content-Security-Policy: default-src 'none'` — une API JSON ne sert aucune
//     ressource active ; tout ce qui s'exécuterait serait injecté.
//   - `Strict-Transport-Security` — protège la PREMIÈRE requête d'une visite
//     ultérieure, celle qu'un attaquant sur le réseau détournerait avant tout
//     échange chiffré.
func TestHardeningHeadersAreAlwaysPresent(t *testing.T) {
	t.Parallel()

	rec := call(middleware.SecurityHeaders(), get(t), okHandler(nil))

	expected := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for header, want := range expected {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, attendu %q", header, got, want)
		}
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy absent")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("HSTS absent du constructeur par DÉFAUT : la protection doit être acquise sans la demander")
	}
}
