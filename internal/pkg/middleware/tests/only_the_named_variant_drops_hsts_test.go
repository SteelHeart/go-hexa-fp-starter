package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestOnlyTheNamedVariantDropsHSTS : seule la variante qui se NOMME renonce à HSTS.
//
// # Ce que ce test protège vraiment
//
// C'était autrefois `SecurityHeaders(secure bool)`. Un booléen de contrôle ne dit
// ni ce qu'il active, ni ce qu'il coûte : `SecurityHeaders(false)` au mauvais
// endroit retirait HSTS en production sans que la relecture le voie.
//
// Désormais la renonciation porte un nom, et ce test vérifie les deux sens — que
// le défaut protège, et que la variante nommée soit la SEULE à ne pas protéger.
// Vérifier un seul des deux laisserait passer la régression qui compte.
//
// La renonciation reste légitime en développement : sur `http://localhost`, HSTS
// inscrirait dans le navigateur une exigence de HTTPS que le poste ne peut pas
// satisfaire, et le développeur perdrait l'accès à son propre serveur jusqu'à
// purger le cache.
func TestOnlyTheNamedVariantDropsHSTS(t *testing.T) {
	t.Parallel()

	withHSTS := call(middleware.SecurityHeaders(), get(t), okHandler(nil))
	without := call(middleware.SecurityHeadersWithoutHSTS(), get(t), okHandler(nil))

	if got := withHSTS.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("SecurityHeaders() doit poser HSTS")
	}
	if got := without.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("SecurityHeadersWithoutHSTS() a posé HSTS: %q", got)
	}

	// Tout le RESTE du durcissement doit survivre à la renonciation : on renonce
	// à HSTS, pas à la protection.
	for _, header := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
		if got := without.Header().Get(header); got == "" {
			t.Errorf("%s perdu par la variante sans HSTS — elle ne renonce qu'à HSTS", header)
		}
	}
}
