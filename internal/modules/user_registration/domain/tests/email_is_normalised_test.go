package tests

import "testing"

// TestEmailIsNormalised : l'adresse est mise en minuscules et débarrassée de ses
// espaces de bordure.
//
// Sans normalisation, `Alice@Example.com ` et `alice@example.com` créeraient DEUX
// comptes. L'utilisateur ne comprendrait pas pourquoi sa connexion échoue, et le
// support découvrirait des doublons impossibles à fusionner proprement.
//
// C'est aussi ce qui rend l'index d'unicité en base réellement efficace : il porte
// sur une forme canonique unique, pas sur une variante de casse.
func TestEmailIsNormalised(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"majuscules":         "Alice@Example.COM",
		"espaces de bordure": "  alice@example.com  ",
		"les deux":           "\tALICE@Example.Com \n",
		"déjà normalisée":    "alice@example.com",
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := emailValide(t, raw).String(); got != "alice@example.com" {
				t.Errorf("NewEmail(%q) = %q, attendu la forme canonique", raw, got)
			}
		})
	}
}
