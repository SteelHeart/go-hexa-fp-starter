package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestWeakPasswordIsRefused : les bornes de mot de passe sont des décisions de
// sécurité, chacune pour une raison différente.
//
//   - **Minimum haut, sans règle de composition.** Une longue phrase de passe
//     résiste mieux qu'un « P@ssw0rd! » court, et les exigences de composition
//     produisent des mots de passe prévisibles — l'utilisateur met une majuscule au
//     début et un « ! » à la fin.
//   - **Maximum.** Argon2id sur une entrée de dix mégaoctets est un déni de service
//     offert : le coût de hachage est payé par le SERVEUR, pas par l'attaquant.
//   - **Répétitivité.** « aaaaaaaaaaaa » fait douze caractères et ne vaut rien.
func TestWeakPasswordIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"vide":                        "",
		"trop court":                  "court123",
		"onze caractères":             "onzecaract1",
		"espaces seulement":           strings.Repeat(" ", 20),
		"trop répétitif":              "aaaaaaaaaaaaaa",
		"quatre caractères distincts": "abcabcabcabcabc",
		"trop long":                   strings.Repeat("phrase-de-passe-", 20),
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := codeDe(t, domain.NewRawPassword(raw)); got != domain.CodeWeakPassword {
				t.Errorf("code = %q, attendu %q", got, domain.CodeWeakPassword)
			}
		})
	}
}
